package dd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"dario.cat/cocoon/cocoontest"
	"dario.cat/cocoon/rt"
)

const pointArgumentMessage = "sketch points must be finite and nonnegative"

type pointValidationCase struct {
	name  string
	bits  uint64
	valid bool
}

func pointValidationCases() []pointValidationCase {
	return []pointValidationCase{
		{"positive zero", 0x0000000000000000, true},
		{"negative zero", 0x8000000000000000, true},
		{"smallest positive subnormal", 0x0000000000000001, true},
		{"smallest negative subnormal", 0x8000000000000001, false},
		{"largest positive subnormal", 0x000fffffffffffff, true},
		{"largest negative subnormal", 0x800fffffffffffff, false},
		{"smallest positive normal", 0x0010000000000000, true},
		{"smallest negative normal", 0x8010000000000000, false},
		{"positive one", 0x3ff0000000000000, true},
		{"negative one", 0xbff0000000000000, false},
		{"largest positive normal", 0x7fefffffffffffff, true},
		{"largest negative normal", 0xffefffffffffffff, false},
		{"positive infinity", 0x7ff0000000000000, false},
		{"negative infinity", 0xfff0000000000000, false},
		{"positive quiet NaN", 0x7ff8000000000000, false},
		{"negative quiet NaN", 0xfff8000000000000, false},
		{"positive signaling NaN", 0x7ff0000000000001, false},
		{"negative signaling NaN", 0xfff0000000000001, false},
		{"positive maximal quiet NaN payload", 0x7fffffffffffffff, false},
		{"negative maximal quiet NaN payload", 0xffffffffffffffff, false},
		{"positive maximal signaling NaN payload", 0x7ff7ffffffffffff, false},
		{"negative maximal signaling NaN payload", 0xfff7ffffffffffff, false},
	}
}

func TestSketchPointValidationEdges(t *testing.T) {
	t.Parallel()
	library := openTest(t, Options{Instances: 1})
	for _, test := range pointValidationCases() {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			sketch, err := library.NewSketch()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if closeErr := sketch.Close(); closeErr != nil {
					t.Error(closeErr)
				}
			})
			if seedErr := sketch.Add(1); seedErr != nil {
				t.Fatal(seedErr)
			}
			beforeCount, beforeEncoding := pointSketchState(t, sketch)
			if beforeCount != 1 {
				t.Fatalf("seed count = %g, want 1", beforeCount)
			}
			value := math.Float64frombits(test.bits)
			err = sketch.Add(value)
			wantCount := beforeCount
			if test.valid {
				if err != nil {
					t.Fatalf("valid bits %016x rejected: %v", test.bits, err)
				}
				wantCount++
			} else {
				checkPointArgument(t, "add", err)
			}
			afterCount, afterEncoding := pointSketchState(t, sketch)
			if afterCount != wantCount {
				t.Fatalf("count = %g, want %g", afterCount, wantCount)
			}
			if !test.valid {
				if !bytes.Equal(beforeEncoding, afterEncoding) {
					t.Fatal("rejected scalar point changed encoded state")
				}
				checkPointArgument(t, "add_many", sketch.AddMany([]float64{2, value, 3}))
				batchCount, batchEncoding := pointSketchState(t, sketch)
				if batchCount != beforeCount || !bytes.Equal(beforeEncoding, batchEncoding) {
					t.Fatal("rejected batch changed state before full validation")
				}
				// Leave an error reply immediately before recovering with success.
				checkPointArgument(t, "add", sketch.Add(value))
			}
			if err := sketch.Add(2); err != nil {
				t.Fatal("valid point failed after domain validation", err)
			}
			if count, err := sketch.Count(); err != nil || count != wantCount+1 {
				t.Fatalf("recovered count = %g, want %g: %v", count, wantCount+1, err)
			}
		})
	}
}

func checkPointArgument(tb testing.TB, operation string, err error) {
	tb.Helper()
	var application *rt.AppError
	if !errors.As(err, &application) || application.Op != operation || application.Status != rt.ErrArg ||
		application.Message != pointArgumentMessage || err.Error() != operation+": "+pointArgumentMessage {
		tb.Fatalf("point argument error = %v, want %s ErrArg with exact message", err, operation)
	}
}

func pointSketchState(tb testing.TB, sketch *Sketch) (count float64, encoded []byte) {
	tb.Helper()
	count, err := sketch.Count()
	if err != nil {
		tb.Fatal(err)
	}
	encoded, err = sketch.Encode()
	if err != nil || len(encoded) == 0 {
		tb.Fatal("invalid encoded sketch", err)
	}
	return count, encoded
}

func TestSketchPointValidationDifferential(t *testing.T) {
	g, reference := newDifferential(t)
	for _, test := range pointValidationCases() {
		t.Run(test.name, func(t *testing.T) {
			status, data := compareOperation(t, g, reference, "cocoon_sketch_new", nil)
			if status != rt.OK || len(data) != 8 {
				t.Fatalf("new status = %d, reply = %x", status, data)
			}
			handle := binary.LittleEndian.Uint64(data)
			arguments := []cocoontest.Arg{{Value: handle}, {Value: math.Float64bits(1)}}
			if seedStatus, reply := compareOperation(t, g, reference, "cocoon_sketch_add", arguments); seedStatus != rt.OK || len(reply) != 0 {
				t.Fatalf("seed status = %d, reply = %x", seedStatus, reply)
			}
			beforeCount, beforeEncoding := pointDifferentialState(t, g, reference, handle)
			if beforeCount != 1 {
				t.Fatalf("seed count = %g, want 1", beforeCount)
			}
			arguments[1].Value = test.bits
			status, reply := compareOperation(t, g, reference, "cocoon_sketch_add", arguments)
			wantCount := beforeCount
			if test.valid {
				if status != rt.OK || len(reply) != 0 {
					t.Fatalf("valid bits %016x: status = %d, reply = %x", test.bits, status, reply)
				}
				wantCount++
			} else {
				if status != rt.ErrArg {
					t.Fatalf("invalid bits %016x: status = %d, want ErrArg", test.bits, status)
				}
				// compareOperation compares error text, not the reference error
				// status. Repeating a rejected call must be nonmutating as well.
				limits := g.instanceLimits()
				status, payload, err := reference.Operation(t.Context(), "cocoon_sketch_add", arguments, limits.MaxInput, limits.MaxOutput)
				if err != nil || status != rt.ErrArg || string(payload) != pointArgumentMessage {
					t.Fatalf("reference rejected point: status = %d, reply = %q: %v", status, payload, err)
				}
			}
			afterCount, afterEncoding := pointDifferentialState(t, g, reference, handle)
			if afterCount != wantCount || !test.valid && !bytes.Equal(beforeEncoding, afterEncoding) {
				t.Fatalf("point changed unexpected state: count = %g, want %g", afterCount, wantCount)
			}
			if !test.valid {
				if status, _ := compareOperation(t, g, reference, "cocoon_sketch_add", arguments); status != rt.ErrArg {
					t.Fatalf("repeated invalid point status = %d", status)
				}
			}
			arguments[1].Value = math.Float64bits(2)
			if status, reply := compareOperation(t, g, reference, "cocoon_sketch_add", arguments); status != rt.OK || len(reply) != 0 {
				t.Fatalf("recovery status = %d, reply = %x", status, reply)
			}
			if count, _ := pointDifferentialState(t, g, reference, handle); count != wantCount+1 {
				t.Fatalf("recovery count = %g, want %g", count, wantCount+1)
			}
			if status, reply := compareOperation(t, g, reference, "cocoon_sketch_close", arguments[:1]); status != rt.OK || len(reply) != 0 {
				t.Fatalf("close status = %d, reply = %x", status, reply)
			}
		})
	}
}

func pointDifferentialState(tb testing.TB, g *guest, reference *cocoontest.Guest, handle uint64) (count float64, encoded []byte) {
	tb.Helper()
	arguments := []cocoontest.Arg{{Value: handle}}
	status, data := compareOperation(tb, g, reference, "cocoon_sketch_count", arguments)
	if status != rt.OK || len(data) != 8 {
		tb.Fatalf("count status = %d, reply = %x", status, data)
	}
	count = math.Float64frombits(binary.LittleEndian.Uint64(data))
	status, encoded = compareOperation(tb, g, reference, "cocoon_sketch_encode", arguments)
	if status != rt.OK || len(encoded) == 0 {
		tb.Fatalf("encode status = %d, reply = %x", status, encoded)
	}
	return count, encoded
}
