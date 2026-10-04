// Package manifest parses and validates Cocoon's declarative API contract.
package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/token"
	"math"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Supported toolchain pins are part of the translator and trap contract.
const (
	RustVersion     = "1.97.0"
	BinaryenVersion = "133"
	Wasm2GoVersion  = "v0.4.16"
)

// Manifest declares API values, ownership, capabilities, limits, and build pins.
type Manifest struct {
	Package      Package      `toml:"package" json:"package"`
	Toolchain    Toolchain    `toml:"toolchain" json:"-"`
	Limits       Limits       `toml:"limits" json:"limits"`
	Functions    []Function   `toml:"func" json:"functions"`
	Resources    []Resource   `toml:"resource" json:"resources"`
	Records      []Record     `toml:"record" json:"records"`
	Sources      []SourcePin  `toml:"source" json:"-"`
	Content      []byte       `toml:"-" json:"-"`
	Capabilities Capabilities `toml:"capabilities" json:"capabilities"`
}

// SourcePin locks a local dependency repository; its content is also hashed.
// Paths are relative to the manifest directory and never enter schema identity.
type SourcePin struct {
	Name     string `toml:"name" json:"name"`
	Path     string `toml:"path" json:"-"`
	Revision string `toml:"revision" json:"revision"`
}

// Package identifies generated Go and Rust packages.
type Package struct {
	Name      string `toml:"name" json:"name"`
	GoImport  string `toml:"go_import" json:"-"`
	RustCrate string `toml:"rust_crate" json:"-"`
}

// Toolchain pins every build tool whose behavior affects generated output.
type Toolchain struct {
	Rust     string `toml:"rust" json:"rust"`
	Binaryen string `toml:"binaryen" json:"binaryen"`
	Wasm2Go  string `toml:"wasm2go" json:"wasm2go"`
}

// Limits accepts explicit binary byte units and a positive instance count.
type Limits struct {
	MaxInput  string `toml:"max_input" json:"max_input"`
	MaxOutput string `toml:"max_output" json:"max_output"`
	MaxMemory string `toml:"max_memory" json:"max_memory"`
	Instances string `toml:"instances" json:"instances"`
}

// Capabilities is the complete supported synchronous host import set.
type Capabilities struct {
	Log    bool `toml:"log" json:"log"`
	Random bool `toml:"random" json:"random"`
	Clock  bool `toml:"clock" json:"clock"`
}

// Param is a named value or record field.
type Param struct {
	Name string `toml:"name" json:"name"`
	Type string `toml:"type" json:"type"`
}

// Function declares a synchronous operation with at most one logical result.
type Function struct {
	Name     string  `toml:"name" json:"name"`
	Returns  string  `toml:"returns" json:"returns"`
	Params   []Param `toml:"params" json:"params"`
	Fallible bool    `toml:"fallible" json:"fallible"`
	Async    bool    `toml:"async" json:"async"`
}

// Resource declares shared ownership, a constructor, destructor, and methods.
type Resource struct {
	Name    string     `toml:"name" json:"name"`
	Copy    string     `toml:"copy" json:"copy"`
	Methods []Function `toml:"method" json:"methods"`
}

// Record declares an ordered set of required typed fields.
type Record struct {
	Name   string  `toml:"name" json:"name"`
	Fields []Param `toml:"fields" json:"fields"`
}

// Parse rejects unknown keys and validates the complete contract before generation.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields().Decode(&m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	m.Content = bytes.Clone(data)
	return &m, nil
}

var (
	identifier = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	crateName  = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	revision   = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// GoName converts snake case and common initialisms into exported Go identifiers.
func GoName(name string) string {
	var output strings.Builder
	for _, part := range strings.Split(name, "_") {
		if part == "" {
			continue
		}
		switch strings.ToLower(part) {
		case "abi", "id", "sql", "http", "json", "utf8", "url":
			output.WriteString(strings.ToUpper(part))
		default:
			output.WriteString(strings.ToUpper(part[:1]))
			output.WriteString(part[1:])
		}
	}
	return output.String()
}

func validName(name string) bool {
	if !identifier.MatchString(name) || !token.IsIdentifier(name) || strings.HasPrefix(name, "cocoon_") {
		return false
	}
	return !slices.Contains(strings.Fields("as async await break const continue crate dyn else enum extern false fn for if impl in let loop match mod move mut pub ref return self Self static struct super trait true type unsafe use where while abstract become box do final macro override priv typeof unsized virtual yield try"), name)
}

// ParseSize accepts bytes or KiB/MiB/GiB without ambiguous decimal units.
func ParseSize(value string) (uint64, error) {
	width := uint64(1)
	for _, unit := range []struct {
		suffix string
		width  uint64
	}{{"KiB", 1 << 10}, {"MiB", 1 << 20}, {"GiB", 1 << 30}, {"B", 1}} {
		if strings.HasSuffix(value, unit.suffix) {
			value = strings.TrimSuffix(value, unit.suffix)
			width = unit.width
			break
		}
	}
	count, err := strconv.ParseUint(value, 10, 64)
	if err != nil || count == 0 || count > math.MaxUint32/width {
		return 0, fmt.Errorf("invalid wasm32 byte limit %q", value)
	}
	return count * width, nil
}

// ByteLimits returns normalized numeric input, output, and memory limits.
func (m *Manifest) ByteLimits() (inputBytes, outputBytes, memoryBytes uint64, err error) {
	input, err := ParseSize(m.Limits.MaxInput)
	if err != nil {
		return 0, 0, 0, err
	}
	output, err := ParseSize(m.Limits.MaxOutput)
	if err != nil {
		return 0, 0, 0, err
	}
	memory, err := ParseSize(m.Limits.MaxMemory)
	if err != nil {
		return 0, 0, 0, err
	}
	if input > memory || output > memory || memory%65536 != 0 {
		return 0, 0, 0, fmt.Errorf("limits exceed memory or memory is not a whole number of pages")
	}
	return input, output, memory, nil
}

// Validate enforces names, value types, ownership, capabilities, and tool pins.
func (m *Manifest) Validate() error {
	if !validName(m.Package.Name) || m.Package.Name != strings.ToLower(m.Package.Name) || !crateName.MatchString(m.Package.RustCrate) {
		return fmt.Errorf("invalid package name or Rust crate")
	}
	if m.Package.GoImport == "" || m.Package.GoImport == "." || path.Clean(m.Package.GoImport) != m.Package.GoImport || strings.HasPrefix(m.Package.GoImport, "/") || strings.ContainsAny(m.Package.GoImport, "\\ :\t\n\r\"'`<>") || slices.Contains(strings.Split(m.Package.GoImport, "/"), "..") {
		return fmt.Errorf("invalid Go import path")
	}
	if m.Toolchain == (Toolchain{}) {
		m.Toolchain = Toolchain{Rust: RustVersion, Binaryen: BinaryenVersion, Wasm2Go: Wasm2GoVersion}
	}
	if m.Toolchain.Rust != RustVersion || m.Toolchain.Binaryen != BinaryenVersion || m.Toolchain.Wasm2Go != Wasm2GoVersion {
		return fmt.Errorf("unsupported toolchain pins")
	}
	if _, _, _, err := m.ByteLimits(); err != nil {
		return err
	}
	if m.Limits.Instances != "gomaxprocs" {
		instances, err := strconv.ParseUint(m.Limits.Instances, 10, 16)
		if err != nil || instances == 0 {
			return fmt.Errorf("invalid instance count")
		}
		m.Limits.Instances = strconv.FormatUint(instances, 10)
	}
	if len(m.Functions)+len(m.Resources) == 0 {
		return fmt.Errorf("manifest has no operations")
	}
	sources := make(map[string]bool)
	for _, source := range m.Sources {
		if !identifier.MatchString(source.Name) || sources[source.Name] || source.Path == "" || strings.ContainsRune(source.Path, 0) || !revision.MatchString(source.Revision) {
			return fmt.Errorf("invalid or duplicate source pin %q", source.Name)
		}
		sources[source.Name] = true
	}
	names := make(map[string]bool)
	for _, name := range strings.Fields("Library Options Open Close API State Error Result Slab RefCell String Bytes Bool Vec I32 U32 I64 U64 F32 F64 SliceI32 SliceU32 SliceI64 SliceU64 SliceF32 SliceF64") {
		names[name] = true
	}
	exports := make(map[string]bool)
	for _, name := range strings.Fields("abi_version schema_hash init in_reserve out trim alloc") {
		exports[name] = true
	}
	claimExport := func(name string) error {
		if exports[name] {
			return fmt.Errorf("duplicate or reserved ABI export %q", name)
		}
		exports[name] = true
		return nil
	}
	claim := func(name string) error {
		if !validName(name) || names[GoName(name)] {
			return fmt.Errorf("invalid, duplicate, or reserved name %q", name)
		}
		names[GoName(name)] = true
		return nil
	}
	for _, record := range m.Records {
		if record.Name != GoName(record.Name) || len(record.Fields) == 0 {
			return fmt.Errorf("records need an exported canonical name and at least one field")
		}
		if err := claim(record.Name); err != nil {
			return err
		}
	}
	for _, resource := range m.Resources {
		if resource.Name != GoName(resource.Name) || resource.Name == "Service" {
			return fmt.Errorf("invalid generated resource name %q", resource.Name)
		}
		if err := claim(resource.Name); err != nil {
			return err
		}
		if err := claim("New" + resource.Name); err != nil {
			return err
		}
	}
	for _, function := range m.Functions {
		if function.Name != strings.ToLower(function.Name) {
			return fmt.Errorf("operation names must be lowercase")
		}
		if err := claim(function.Name); err != nil {
			return err
		}
		if err := claimExport(function.Name); err != nil {
			return err
		}
		if err := m.validateFunction(function, ""); err != nil {
			return err
		}
	}
	for _, record := range m.Records {
		if err := m.validateParams(record.Fields); err != nil {
			return err
		}
	}
	for index := range m.Resources {
		resource := &m.Resources[index]
		if resource.Copy == "" {
			resource.Copy = "shared"
		}
		if resource.Copy != "shared" {
			return fmt.Errorf("unsupported ownership %q", resource.Copy)
		}
		methods := make(map[string]bool)
		for _, method := range resource.Methods {
			if err := claimExport(strings.ToLower(resource.Name) + "_" + method.Name); err != nil {
				return err
			}
			if methods[GoName(method.Name)] || !validName(method.Name) || method.Name != strings.ToLower(method.Name) {
				return fmt.Errorf("invalid or duplicate method %q", method.Name)
			}
			methods[GoName(method.Name)] = true
			if err := m.validateFunction(method, resource.Name); err != nil {
				return err
			}
			if method.Name == "new" && method.Returns != resource.Name {
				return fmt.Errorf("resource constructor must return %s", resource.Name)
			}
			if method.Name == "close" && (method.Returns != "" || len(method.Params) != 0) {
				return fmt.Errorf("resource destructor must take no parameters and return no value")
			}
		}
		if !methods["New"] || !methods["Close"] {
			return fmt.Errorf("resource %s requires new and close", resource.Name)
		}
	}
	return m.validateRecordCycles()
}

// Variable reports whether a type occupies a variable-size input buffer region.
func (m *Manifest) Variable(name string) bool {
	return name == "string" || name == "bytes" || strings.HasPrefix(name, "[]") || slices.ContainsFunc(m.Records, func(record Record) bool { return record.Name == name })
}

func (m *Manifest) valueType(name string) bool {
	if slices.Contains([]string{"i32", "u32", "i64", "u64", "f32", "f64", "bool", "string", "bytes", "[]i32", "[]u32", "[]i64", "[]u64", "[]f32", "[]f64"}, name) {
		return true
	}
	return slices.ContainsFunc(m.Records, func(record Record) bool { return record.Name == name })
}

func (m *Manifest) validateParams(params []Param) error {
	names := make(map[string]bool)
	for _, param := range params {
		if !validName(param.Name) || names[GoName(param.Name)] || !m.valueType(param.Type) {
			return fmt.Errorf("invalid or duplicate parameter %q of type %q", param.Name, param.Type)
		}
		names[GoName(param.Name)] = true
	}
	return nil
}

func (m *Manifest) validateFunction(function Function, resource string) error {
	if function.Async {
		return fmt.Errorf("async is reserved for the next milestone")
	}
	if function.Returns != "" && !m.valueType(function.Returns) && (function.Name != "new" || function.Returns != resource) {
		return fmt.Errorf("unsupported return type %q", function.Returns)
	}
	return m.validateParams(function.Params)
}

func (m *Manifest) validateRecordCycles() error {
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var visit func(name string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("recursive record %s", name)
		}
		if visited[name] {
			return nil
		}
		visiting[name] = true
		for _, record := range m.Records {
			if record.Name == name {
				for _, field := range record.Fields {
					if err := visit(field.Type); err != nil {
						return err
					}
				}
			}
		}
		visiting[name] = false
		visited[name] = true
		return nil
	}
	for _, record := range m.Records {
		if err := visit(record.Name); err != nil {
			return err
		}
	}
	return nil
}

// SchemaHash derives a stable identifier from normalized semantic declarations.
func (m *Manifest) SchemaHash() (schema uint64, full string, err error) {
	input, output, memory, err := m.ByteLimits()
	if err != nil {
		return 0, "", err
	}
	normalized := *m
	normalized.Limits.MaxInput = strconv.FormatUint(input, 10)
	normalized.Limits.MaxOutput = strconv.FormatUint(output, 10)
	normalized.Limits.MaxMemory = strconv.FormatUint(memory, 10)
	normalized.Functions = append([]Function{}, m.Functions...)
	normalized.Records = append([]Record{}, m.Records...)
	normalized.Resources = append([]Resource{}, m.Resources...)
	for index := range normalized.Functions {
		normalized.Functions[index].Params = append([]Param{}, normalized.Functions[index].Params...)
	}
	for index := range normalized.Records {
		normalized.Records[index].Fields = append([]Param{}, normalized.Records[index].Fields...)
	}
	slices.SortFunc(normalized.Functions, func(a, b Function) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(normalized.Records, func(a, b Record) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(normalized.Resources, func(a, b Resource) int { return strings.Compare(a.Name, b.Name) })
	for index := range normalized.Resources {
		normalized.Resources[index].Methods = slices.Clone(normalized.Resources[index].Methods)
		for method := range normalized.Resources[index].Methods {
			normalized.Resources[index].Methods[method].Params = append([]Param{}, normalized.Resources[index].Methods[method].Params...)
		}
		slices.SortFunc(normalized.Resources[index].Methods, func(a, b Function) int { return strings.Compare(a.Name, b.Name) })
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return 0, "", fmt.Errorf("hash schema: %w", err)
	}
	digest := sha256.Sum256(data)
	return binary.LittleEndian.Uint64(digest[:8]), hex.EncodeToString(digest[:]), nil
}
