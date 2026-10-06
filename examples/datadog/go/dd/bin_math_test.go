package dd

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/darccio/cocoon/cocoontest"
	"github.com/darccio/cocoon/rt"
)

type sketchWireField struct {
	data   []byte
	number uint64
	value  uint64
	wire   uint64
}

// Read only the protobuf wire types used by the pinned sketch schema.
func sketchWireFields(tb testing.TB, data []byte) []sketchWireField {
	tb.Helper()
	var fields []sketchWireField
	for len(data) != 0 {
		tag, size := binary.Uvarint(data)
		if size <= 0 || tag>>3 == 0 {
			tb.Fatal("invalid sketch protobuf tag")
		}
		data = data[size:]
		field := sketchWireField{number: tag >> 3, wire: tag & 7}
		switch field.wire {
		case 0:
			field.value, size = binary.Uvarint(data)
			if size <= 0 {
				tb.Fatal("invalid sketch protobuf varint")
			}
		case 1:
			size = 8
		case 2:
			length, prefix := binary.Uvarint(data)
			if prefix <= 0 || length > uint64(len(data)-prefix) {
				tb.Fatal("invalid sketch protobuf length")
			}
			data = data[prefix:]
			size = int(length) // #nosec G115 -- Length is bounded by the available slice.
		default:
			tb.Fatalf("unsupported sketch protobuf wire type %d", field.wire)
		}
		if size > len(data) {
			tb.Fatal("truncated sketch protobuf field")
		}
		field.data = data[:size]
		fields = append(fields, field)
		data = data[size:]
	}
	return fields
}

type singletonSketchBin struct {
	index     int64
	zeroCount uint64
	hasBin    bool
}

// The first nonzero packed count, not the store offset alone, locates the bin.
func readSingletonSketchBin(tb testing.TB, encoded []byte) singletonSketchBin {
	tb.Helper()
	var result singletonSketchBin
	var store []byte
	for _, field := range sketchWireFields(tb, encoded) {
		switch field.number {
		case 2:
			if field.wire != 2 {
				tb.Fatal("invalid positive store wire type")
			}
			store = field.data
		case 4:
			if field.wire != 1 {
				tb.Fatal("invalid zero count wire type")
			}
			result.zeroCount = binary.LittleEndian.Uint64(field.data)
		}
	}
	var offset, countIndex, firstIndex int64
	for _, field := range sketchWireFields(tb, store) {
		switch field.number {
		case 1:
			tb.Fatal("unexpected sparse counts in pinned sketch")
		case 2:
			if field.wire != 2 || len(field.data)%8 != 0 {
				tb.Fatal("invalid packed sketch counts")
			}
			for len(field.data) != 0 {
				count := math.Float64frombits(binary.LittleEndian.Uint64(field.data))
				if count != 0 && !result.hasBin {
					result.hasBin = true
					firstIndex = countIndex
				}
				countIndex++
				field.data = field.data[8:]
			}
		case 3:
			if field.wire != 0 || field.value > math.MaxUint32 {
				tb.Fatal("invalid sketch bin offset")
			}
			offset = int64(field.value>>1) ^ -int64(field.value&1) // #nosec G115 -- The zigzag input is bounded to uint32.
		}
	}
	result.index = offset + firstIndex
	if result.index < math.MinInt32 || result.index > math.MaxInt32 {
		tb.Fatal("sketch bin index overflow")
	}
	return result
}

func binOracleOperation(tb testing.TB, reference *cocoontest.Guest, limits rt.Limits, operation string, args []cocoontest.Arg) []byte {
	tb.Helper()
	status, data, err := reference.Operation(tb.Context(), operation, args, limits.MaxInput, limits.MaxOutput)
	if err != nil || status != rt.OK {
		tb.Fatalf("oracle %s: status=%d, error=%v", operation, status, err)
	}
	return data
}

func binOracleState(tb testing.TB, reference *cocoontest.Guest, limits rt.Limits, points []float64) (countBits uint64, encoding []byte) {
	tb.Helper()
	data := binOracleOperation(tb, reference, limits, "cocoon_sketch_new", nil)
	if len(data) != 8 {
		tb.Fatal("invalid oracle sketch handle")
	}
	handle := binary.LittleEndian.Uint64(data)
	defer binOracleOperation(tb, reference, limits, "cocoon_sketch_close", []cocoontest.Arg{{Value: handle}})
	for _, point := range points {
		binOracleOperation(tb, reference, limits, "cocoon_sketch_add", []cocoontest.Arg{{Value: handle}, {Value: math.Float64bits(point)}})
	}
	count := binOracleOperation(tb, reference, limits, "cocoon_sketch_count", []cocoontest.Arg{{Value: handle}})
	if len(count) != 8 {
		tb.Fatal("invalid oracle sketch count")
	}
	encoded := binOracleOperation(tb, reference, limits, "cocoon_sketch_encode", []cocoontest.Arg{{Value: handle}})
	return binary.LittleEndian.Uint64(count), encoded
}

// Adjacent binary64 values were located by monotone bit-pattern search against
// the unchanged pinned Wasm in wazero, never by a host logarithm approximation.
var sketchBinTransitions = []struct {
	name          string
	lower, upper  uint64
	before, after int64
	beforeHasBin  bool
}{
	{"very small negative bins", 0x06245df348a79e42, 0x06245df348a79e43, -40001, -40000, true},
	{"negative bins", 0x3d718047c1411806, 0x3d718047c1411807, -445, -444, true},
	{"negative to zero bin", 0x3e10aff5a327ce7b, 0x3e10aff5a327ce7c, -1, 0, true},
	{"zero to positive bin", 0x3e10f2b579b46dad, 0x3e10f2b579b46dae, 0, 1, true},
	{"below one", 0x3fefc0bd88a0f1c9, 0x3fefc0bd88a0f1ca, 1337, 1338, true},
	{"above one", 0x3ff01fe03f61bac8, 0x3ff01fe03f61bac9, 1338, 1339, true},
	{"large positive bins", 0x5a055c8cfd963fb9, 0x5a055c8cfd963fba, 19999, 20000, true},
	{"near largest input", 0x7fc45cfcb8542291, 0x7fc45cfcb8542292, 46999, 47000, true},
	{"zero-count threshold", 0x00103fffffffffff, 0x0010400000000000, 0, -44352, false},
}

func assertBinMathState(tb testing.TB, library *Library, reference *cocoontest.Guest, limits rt.Limits, points []float64, batch bool) []byte {
	tb.Helper()
	// Compare both public paths to scalar execution of the original Wasm.
	wantCount, wantEncoding := binOracleState(tb, reference, limits, points)
	sketch, err := library.NewSketch()
	if err != nil {
		tb.Fatal(err)
	}
	defer func() {
		if err := sketch.Close(); err != nil {
			tb.Error(err)
		}
	}()
	if batch {
		if err := sketch.AddMany(points); err != nil {
			tb.Fatal(err)
		}
	} else {
		for _, point := range points {
			if err := sketch.Add(point); err != nil {
				tb.Fatal(err)
			}
		}
	}
	count, encoded := pointSketchState(tb, sketch)
	if math.Float64bits(count) != wantCount || wantCount != math.Float64bits(float64(len(points))) {
		tb.Fatalf("count bits=%016x, oracle=%016x, points=%d", math.Float64bits(count), wantCount, len(points))
	}
	if !bytes.Equal(encoded, wantEncoding) {
		tb.Fatalf("batch=%v: protobuf differs from unchanged Wasm for points %v", batch, points)
	}
	return encoded
}

func TestSketchBinMathBoundaries(t *testing.T) {
	// Cases share one oracle instance and therefore run serially.
	g, reference := newDifferential(t)
	limits := g.instanceLimits()
	library := openTest(t, Options{Instances: 1})
	for _, transition := range sketchBinTransitions {
		t.Run(transition.name, func(t *testing.T) {
			if transition.upper != transition.lower+1 ||
				math.Float64bits(math.Nextafter(math.Float64frombits(transition.lower), math.Inf(1))) != transition.upper {
				t.Fatal("frozen boundary values are not adjacent")
			}
			// Fresh singleton sketches prevent the 2048-bin collapse policy
			// from hiding a wrong boundary under previously inserted values.
			for _, bits := range []uint64{transition.lower - 1, transition.lower, transition.upper, transition.upper + 1} {
				point := math.Float64frombits(bits)
				encoded := assertBinMathState(t, library, reference, limits, []float64{point}, false)
				state := readSingletonSketchBin(t, encoded)
				index, hasBin := transition.after, true
				if bits <= transition.lower {
					index, hasBin = transition.before, transition.beforeHasBin
				}
				zeroCount := uint64(0)
				if !hasBin {
					zeroCount = math.Float64bits(1)
				}
				if state.index != index || state.hasBin != hasBin || state.zeroCount != zeroCount {
					t.Fatalf("input bits=%016x: bin=%+v, want index=%d hasBin=%v zeroCount=%016x", bits, state, index, hasBin, zeroCount)
				}
				assertBinMathState(t, library, reference, limits, []float64{point}, true)
			}
			points := []float64{
				math.Float64frombits(transition.lower),
				math.Float64frombits(transition.upper),
				math.Nextafter(math.Float64frombits(transition.upper), math.Inf(1)),
			}
			assertBinMathState(t, library, reference, limits, points, false)
			assertBinMathState(t, library, reference, limits, points, true)
		})
	}
	for _, points := range [][]float64{
		{0, math.Copysign(0, -1), math.SmallestNonzeroFloat64, math.Float64frombits(0x0010000000000000), 1e-12, 1e-11, math.Float64frombits(0x3e10aff5a327ce7c)},
		{math.Float64frombits(0x3e10aff5a327ce7c), 1e-11, 1e-12, math.Float64frombits(0x0010000000000000), math.SmallestNonzeroFloat64, math.Copysign(0, -1), 0},
	} {
		assertBinMathState(t, library, reference, limits, points, false)
		assertBinMathState(t, library, reference, limits, points, true)
	}
}

func TestSketchBinReaderUsesFirstNonzeroCount(t *testing.T) {
	t.Parallel()
	counts := make([]byte, 24)
	binary.LittleEndian.PutUint64(counts[16:], math.Float64bits(1))
	store := append([]byte{0x12, 0x18}, counts...)
	store = append(store, 0x18, 0x05) // Store offset -3, zigzag encoded after the counts.
	encoded := append([]byte{0x12, 0x1c}, store...)
	state := readSingletonSketchBin(t, encoded)
	if !state.hasBin || state.index != -1 || state.zeroCount != 0 {
		t.Fatalf("reader used the offset instead of the first populated bin: %+v", state)
	}
}
