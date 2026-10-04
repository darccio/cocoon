// Package wasmbin reads bounded WebAssembly ABI sections without interpreter internals.
// Read validates structure; the build pipeline also runs Binaryen's code validator.
package wasmbin

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

// Value is a WebAssembly numeric value type.
type Value byte

// Supported ABI value types use WebAssembly's standard encodings.
const (
	I32 Value = 0x7f
	I64 Value = 0x7e
	F32 Value = 0x7d
	F64 Value = 0x7c
)

// Type is a typed function signature.
type Type struct {
	Params  []Value
	Results []Value
}

// Memory describes one explicitly bounded, unshared wasm32 linear memory.
type Memory struct {
	Min uint32
	Max uint32
}

// Import preserves the namespace and name alongside its function signature index.
type Import struct {
	Module string
	Name   string
	Type   uint32
	Kind   byte
}

// Export identifies a function or memory by its module index.
type Export struct {
	Name  string
	Index uint32
	Kind  byte
}

// Module contains all ABI metadata needed for manifest verification.
type Module struct {
	Types     []Type
	Imports   []Import
	Exports   []Export
	Functions []uint32
	Memories  []Memory
	Features  []string
	HasStart  bool
}

type cursor struct {
	err  error
	data []byte
	pos  int
}

func (c *cursor) fail(message string) {
	if c.err == nil {
		c.err = fmt.Errorf("invalid wasm at byte %d: %s", c.pos, message)
	}
}

func (c *cursor) take(size uint32) []byte {
	if uint64(size) > uint64(len(c.data)-c.pos) { // #nosec G115 -- Cursor advances only inside checked take, so remaining is nonnegative.
		c.fail("truncated section")
		return nil
	}
	start := c.pos
	c.pos += int(size)
	return c.data[start:c.pos]
}

func (c *cursor) byte() byte {
	data := c.take(1)
	if len(data) == 0 {
		return 0
	}
	return data[0]
}

func (c *cursor) u32() uint32 {
	var value uint32
	for index := range 5 {
		b := c.byte()
		if index == 4 && b > 15 {
			c.fail("u32 LEB overflow")
			return 0
		}
		value |= uint32(b&0x7f) << (index * 7)
		if b&0x80 == 0 {
			return value
		}
	}
	c.fail("unterminated LEB")
	return 0
}

func (c *cursor) count() uint32 {
	count := c.u32()
	if count > 100000 || uint64(count) > uint64(len(c.data)-c.pos) { // #nosec G115 -- Cursor position never exceeds data length.
		c.fail("unbounded vector count")
		return 0
	}
	return count
}

func (c *cursor) name() string {
	data := c.take(c.u32())
	if !utf8.Valid(data) {
		c.fail("non-UTF8 name")
	}
	return string(data)
}

func (c *cursor) values() []Value {
	count := c.count()
	values := make([]Value, count)
	for index := range values {
		value := Value(c.byte())
		switch value {
		case I32, I64, F32, F64:
		default:
			c.fail("unsupported function value type")
		}
		values[index] = value
	}
	return values
}

func (c *cursor) memory() Memory {
	flags := c.u32()
	if flags != 1 {
		c.fail("memory requires a maximum and must be unshared wasm32")
		return Memory{}
	}
	memory := Memory{Min: c.u32(), Max: c.u32()}
	if memory.Min > memory.Max || memory.Max > 65536 {
		c.fail("invalid memory limits")
	}
	return memory
}

func (c *cursor) table() {
	if c.byte() != 0x70 {
		c.fail("unsupported table type")
	}
	flags := c.u32()
	if flags > 1 {
		c.fail("unsupported table limits")
		return
	}
	minimum := c.u32()
	if flags == 1 && c.u32() < minimum {
		c.fail("invalid table limits")
	}
}

// Read rejects malformed ABI sections, duplicate names, and unsupported declarations.
func Read(data []byte) (*Module, error) {
	if len(data) < 8 || !bytes.Equal(data[:8], []byte{0, 'a', 's', 'm', 1, 0, 0, 0}) {
		return nil, fmt.Errorf("invalid wasm header")
	}
	c := &cursor{data: data, pos: 8}
	module := new(Module)
	seen := make(map[byte]bool)
	last := byte(0)
	var declared, bodies uint32
	for c.pos < len(data) && c.err == nil {
		id := c.byte()
		size := c.u32()
		section := &cursor{data: c.take(size)}
		if c.err != nil {
			break
		}
		if id > 12 {
			c.fail("unsupported section")
			break
		}
		if id != 0 {
			// Data count precedes code and data despite its numerical section ID.
			order := id
			if id == 12 {
				order = 10
			} else if id >= 10 {
				order = id + 1
			}
			if seen[id] || order < last {
				c.fail("duplicate or unordered section")
				break
			}
			seen[id] = true
			last = order
		}
		switch id {
		case 0:
			name := section.name()
			if name == "target_features" {
				for range section.count() {
					prefix := section.byte()
					feature := section.name()
					if prefix == '+' {
						module.Features = append(module.Features, feature)
					} else if prefix != '-' {
						section.fail("invalid target feature prefix")
					}
				}
			} else {
				section.pos = len(section.data)
			}
		case 1:
			for range section.count() {
				if section.byte() != 0x60 {
					section.fail("unsupported type declaration")
					break
				}
				module.Types = append(module.Types, Type{Params: section.values(), Results: section.values()})
			}
		case 2:
			for range section.count() {
				imported := Import{Module: section.name(), Name: section.name(), Kind: section.byte()}
				switch imported.Kind {
				case 0:
					imported.Type = section.u32()
					module.Functions = append(module.Functions, imported.Type)
				case 1:
					section.table()
				case 2:
					module.Memories = append(module.Memories, section.memory())
				case 3:
					value := Value(section.byte())
					if value != I32 && value != I64 && value != F32 && value != F64 {
						section.fail("unsupported imported global")
					}
					if section.byte() > 1 {
						section.fail("invalid global mutability")
					}
				default:
					section.fail("unsupported import kind")
				}
				module.Imports = append(module.Imports, imported)
			}
		case 3:
			declared = section.count()
			for range declared {
				module.Functions = append(module.Functions, section.u32())
			}
		case 4:
			for range section.count() {
				section.table()
			}
		case 5:
			for range section.count() {
				module.Memories = append(module.Memories, section.memory())
			}
		case 7:
			for range section.count() {
				module.Exports = append(module.Exports, Export{Name: section.name(), Kind: section.byte(), Index: section.u32()})
			}
		case 8:
			module.HasStart = true
			if section.u32() >= uint32(len(module.Functions)) {
				section.fail("invalid start index")
			} // #nosec G115 -- Vector counts are bounded.
		case 10:
			bodies = section.count()
			for range bodies {
				body := section.take(section.u32())
				if len(body) == 0 || body[len(body)-1] != 0x0b {
					section.fail("invalid function body")
				}
			}
		case 12:
			section.u32()
		default:
			// Global, element, and data initializer code is validated by Binaryen.
			section.pos = len(section.data)
		}
		if section.err != nil {
			return nil, section.err
		}
		if section.pos != len(section.data) {
			return nil, fmt.Errorf("trailing bytes in wasm section %d", id)
		}
	}
	if c.err != nil {
		return nil, c.err
	}
	if declared != bodies {
		return nil, fmt.Errorf("function/code count mismatch")
	}
	for _, index := range module.Functions {
		if uint64(index) >= uint64(len(module.Types)) {
			return nil, fmt.Errorf("invalid function type index")
		}
	}
	names := make(map[string]bool)
	for _, exported := range module.Exports {
		if names[exported.Name] {
			return nil, fmt.Errorf("duplicate export %q", exported.Name)
		}
		names[exported.Name] = true
		if exported.Kind == 0 && uint64(exported.Index) >= uint64(len(module.Functions)) {
			return nil, fmt.Errorf("invalid exported function index")
		}
		if exported.Kind == 2 && uint64(exported.Index) >= uint64(len(module.Memories)) {
			return nil, fmt.Errorf("invalid exported memory index")
		}
		if exported.Kind > 3 {
			return nil, fmt.Errorf("unsupported export kind")
		}
	}
	return module, nil
}

// Signature resolves a function export's exact numeric signature.
func (m *Module) Signature(exported Export) (Type, error) {
	if exported.Kind != 0 || uint64(exported.Index) >= uint64(len(m.Functions)) {
		return Type{}, fmt.Errorf("export %s is not a function", exported.Name)
	}
	return m.Types[m.Functions[exported.Index]], nil
}
