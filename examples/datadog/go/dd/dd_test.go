package dd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"sync"
	"testing"

	"dario.cat/cocoon/cocoontest"
	"dario.cat/cocoon/rt"
)

func openTest(tb testing.TB, options Options) *Library {
	tb.Helper()
	library, openErr := Open(options)
	if openErr != nil {
		tb.Fatal(openErr)
	}
	tb.Cleanup(func() {
		if err := library.Close(); err != nil {
			tb.Error(err)
		}
	})
	return library
}

func TestDecodedBytesOwnTheirStorage(t *testing.T) {
	t.Parallel()
	data := []byte{1, 2, 3}
	decoded, err := decodeBytes(data, 3)
	if err != nil {
		t.Fatal(err)
	}
	clear(data)
	if !bytes.Equal(decoded, []byte{1, 2, 3}) {
		t.Fatal("byte decoder retained borrowed memory", decoded)
	}
	if _, err := decodeBytes(data, 2); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal("byte decoder skipped its output limit", err)
	}
}

func TestScalarReplyDecodeDoesNotAllocate(t *testing.T) {
	library := openTest(t, Options{Instances: 1})
	sketch, sketchErr := library.NewSketch()
	if sketchErr != nil {
		t.Fatal(sketchErr)
	}
	t.Cleanup(func() {
		if closeErr := sketch.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	if addErr := sketch.Add(1); addErr != nil {
		t.Fatal(addErr)
	}
	allocations := testing.AllocsPerRun(100, func() {
		count, countErr := sketch.Count()
		if countErr != nil || count != 1 {
			t.Fatal("incorrect scalar result", count, countErr)
		}
	})
	if allocations != 0 {
		t.Fatalf("scalar reply allocated %g times", allocations)
	}
}

func TestSQLTracesAndSketch(t *testing.T) {
	t.Parallel()
	library := openTest(t, Options{Instances: 1})
	query := "SELECT * FROM accounts WHERE id = 424242 AND secret = 'private-value'"
	sql, sqlErr := library.ObfuscateSQL(t.Context(), query)
	if sqlErr != nil || strings.Contains(sql, "424242") || strings.Contains(sql, "private-value") || !strings.Contains(sql, "?") {
		t.Fatalf("SQL: %q %v", sql, sqlErr)
	}
	for _, payload := range [][]byte{nil, {0xc1}, {0x90, 0}} {
		var application *rt.AppError
		if _, err := library.ObfuscateTraces(t.Context(), payload); !errors.As(err, &application) {
			t.Fatalf("malformed trace: %v", err)
		}
	}
	traces, tracesErr := library.ObfuscateTraces(t.Context(), tracePayload(2))
	if tracesErr != nil || bytes.Contains(traces, []byte("424242")) || bytes.Contains(traces, []byte("private-value")) {
		t.Fatalf("trace secrets survived: %v", tracesErr)
	}
	if empty, err := library.ObfuscateTraces(t.Context(), []byte{0x90}); err != nil || !bytes.Equal(empty, []byte{0x90}) {
		t.Fatal(empty, err)
	}
	sketch, sketchErr := library.NewSketch()
	if sketchErr != nil {
		t.Fatal(sketchErr)
	}
	t.Cleanup(func() {
		if err := sketch.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := sketch.Add(1); err != nil {
		t.Fatal(err)
	}
	for _, value := range []float64{-1, math.NaN(), math.Inf(1)} {
		if err := sketch.AddMany([]float64{2, value}); err == nil {
			t.Fatal("invalid batch accepted")
		}
		if count, err := sketch.Count(); err != nil || count != 1 {
			t.Fatal(count, err)
		}
	}
	if err := sketch.AddMany([]float64{0, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if count, err := sketch.Count(); err != nil || count != 4 {
		t.Fatal(count, err)
	}
	encoded, encodeErr := sketch.Encode()
	if encodeErr != nil || len(encoded) == 0 {
		t.Fatal(encoded, encodeErr)
	}
	if err := sketch.Add(4); err != nil {
		t.Fatal(err)
	}
	again, err := sketch.Encode()
	if err != nil || bytes.Equal(encoded, again) {
		t.Fatal("encoded reply was not independent", err)
	}
}

func TestBatchLimitAndConcurrentShutdown(t *testing.T) {
	t.Parallel()
	library := openTest(t, Options{MaxInput: 16, MaxOutput: 64, Instances: 2})
	sketch, err := library.NewSketch()
	if err != nil {
		t.Fatal(err)
	}
	if err := sketch.AddMany([]float64{1, 2, 3}); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal(err)
	}
	if count, err := sketch.Count(); err != nil || count != 0 {
		t.Fatal(count, err)
	}
	if err := sketch.AddMany([]float64{1, 2}); err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 20 {
				if err := sketch.Add(1); err != nil && !errors.Is(err, rt.ErrClosed) {
					t.Error(err)
					return
				}
			}
		})
	}
	if err := library.Close(); err != nil {
		t.Fatal(err)
	}
	workers.Wait()
	if _, err := library.NewSketch(); !errors.Is(err, rt.ErrClosed) {
		t.Fatal(err)
	}
	if _, err := library.ObfuscateSQL(t.Context(), "SELECT 1"); !errors.Is(err, rt.ErrClosed) {
		t.Fatal(err)
	}
}

func TestPositiveDatadogDifferential(t *testing.T) {
	t.Parallel()
	g, reference := newDifferential(t)
	for _, query := range []string{"SELECT 42", "SELECT * FROM t WHERE id IN (1,2,3)", "SELECT 'quoted\x00世界'", "/* comment */ SELECT true"} {
		compareOperation(t, g, reference, "cocoon_obfuscate_sql", []cocoontest.Arg{{Buffer: true, Bytes: []byte(query)}})
	}
	compareOperation(t, g, reference, "cocoon_obfuscate_traces", []cocoontest.Arg{{Buffer: true, Bytes: tracePayload(10)}})
	_, data := compareOperation(t, g, reference, "cocoon_sketch_new", nil)
	handle := binary.LittleEndian.Uint64(data)
	for _, value := range []float64{0, 1, 1e-12, 1e12, -1, math.NaN(), math.Inf(1)} {
		compareOperation(t, g, reference, "cocoon_sketch_add", []cocoontest.Arg{{Value: handle}, {Value: math.Float64bits(value)}})
	}
	values, err := encodeSliceF64([]float64{2, 3, 4}, generatedInputLimit)
	if err != nil {
		t.Fatal(err)
	}
	compareOperation(t, g, reference, "cocoon_sketch_add_many", []cocoontest.Arg{{Value: handle}, {Buffer: true, Bytes: values}})
	compareOperation(t, g, reference, "cocoon_sketch_count", []cocoontest.Arg{{Value: handle}})
	compareOperation(t, g, reference, "cocoon_sketch_encode", []cocoontest.Arg{{Value: handle}})
	compareOperation(t, g, reference, "cocoon_sketch_close", []cocoontest.Arg{{Value: handle}})
	compareOperation(t, g, reference, "cocoon_sketch_add", []cocoontest.Arg{{Value: handle}, {Value: math.Float64bits(1)}})
}

func TestResourceAdmissionWithoutContext(t *testing.T) {
	t.Parallel()
	var absent *Library
	if _, err := absent.NewSketch(); !errors.Is(err, rt.ErrClosed) {
		t.Fatal("nil library constructor admitted", err)
	}
	library := openTest(t, Options{Instances: 1})
	sketch, err := library.NewSketch()
	if err != nil {
		t.Fatal(err)
	}
	if err := library.Close(); err != nil {
		t.Fatal(err)
	}
	if err := sketch.Add(1); !errors.Is(err, rt.ErrClosed) {
		t.Fatal("resource method admitted after library close", err)
	}
	if _, err := sketch.Count(); !errors.Is(err, rt.ErrClosed) {
		t.Fatal("scalar method admitted after library close", err)
	}
	if err := sketch.Close(); !errors.Is(err, rt.ErrClosed) {
		t.Fatal("resource destructor lost terminal instance state", err)
	}
}

func TestUnitRepliesAfterDataAndErrors(t *testing.T) {
	t.Parallel()
	g, reference := newDifferential(t)
	_, data := compareOperation(t, g, reference, "cocoon_sketch_new", nil)
	handle := binary.LittleEndian.Uint64(data)
	arguments := []cocoontest.Arg{{Value: handle}, {Value: math.Float64bits(1)}}
	for _, value := range []float64{-1, math.NaN(), math.Inf(1)} {
		// A constructor or scalar reply leaves nonempty output in both guests.
		if status, reply := compareOperation(t, g, reference, "cocoon_sketch_add", arguments); status != rt.OK || len(reply) != 0 {
			t.Fatalf("unit reply after data: status=%d, reply=%x", status, reply)
		}
		arguments[1].Value = math.Float64bits(value)
		if status, _ := compareOperation(t, g, reference, "cocoon_sketch_add", arguments); status != rt.ErrArg {
			t.Fatalf("invalid point status=%d", status)
		}
		arguments[1].Value = math.Float64bits(1)
		if status, reply := compareOperation(t, g, reference, "cocoon_sketch_add", arguments); status != rt.OK || len(reply) != 0 {
			t.Fatalf("unit reply after error: status=%d, reply=%x", status, reply)
		}
		compareOperation(t, g, reference, "cocoon_sketch_count", arguments[:1])
	}
	if status, reply := compareOperation(t, g, reference, "cocoon_sketch_close", arguments[:1]); status != rt.OK || len(reply) != 0 {
		t.Fatalf("close reply after data: status=%d, reply=%x", status, reply)
	}
}

// Encode an independently authored v0.4 MessagePack payload without another
// dependency. One trace contains count SQL spans, including a SQL-query tag.
func tracePayload(count int) []byte {
	if count < 0 || count > 65535 {
		panic("invalid test trace count")
	}
	data := []byte{0x91, 0xdc, byte(count >> 8), byte(count)} // #nosec G115 -- Checked uint16 count encoded in big endian.
	for range count {
		data = append(data, 0x84)
		for _, pair := range [][2]string{{"type", "sql"}, {"resource", "SELECT * FROM t WHERE id = 424242 AND password = 'private-value'"}, {"name", "query"}} {
			data = msgpackString(data, pair[0])
			data = msgpackString(data, pair[1])
		}
		data = msgpackString(data, "meta")
		data = append(data, 0x81)
		data = msgpackString(data, "sql.query")
		data = msgpackString(data, "SELECT 424242, 'private-value'")
	}
	return data
}

func msgpackString(data []byte, value string) []byte {
	if len(value) > 255 {
		panic("test string too long")
	}
	data = append(data, 0xd9, byte(len(value))) // #nosec G115 -- Explicit uint8 length check above.
	return append(data, value...)
}
