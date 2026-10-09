package rt_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"dario.cat/cocoon/rt"
)

func TestRecordCodec(t *testing.T) {
	t.Parallel()
	fields := []rt.Field{{ID: 1, Data: []byte("a=b\nvalue")}, {ID: 2, Data: []byte{0, 255}}}
	encoded, err := rt.EncodeRecord(fields, 128)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := rt.DecodeRecord(encoded, 2, 128)
	if err != nil {
		t.Fatal(err)
	}
	for index, field := range fields {
		if !bytes.Equal(decoded[index], field.Data) || cap(decoded[index]) != len(decoded[index]) {
			t.Fatal("record round trip failed")
		}
	}
	if _, limitErr := rt.EncodeRecord(fields, 10); !errors.Is(limitErr, rt.ErrTooLarge) {
		t.Fatal("limit ignored")
	}
	if _, orderErr := rt.EncodeRecord([]rt.Field{{ID: 2}}, 128); !errors.Is(orderErr, rt.ErrProtocol) {
		t.Fatal("unknown field accepted")
	}
	for index := range encoded {
		if _, truncateErr := rt.DecodeRecord(encoded[:index], 2, 128); truncateErr == nil {
			t.Fatalf("truncation at %d accepted", index)
		}
	}
	bad := bytes.Clone(encoded)
	binary.LittleEndian.PutUint32(bad[4:], 2)
	for _, data := range [][]byte{bad, append(bytes.Clone(encoded), 0), {0, 0, 0, 0}} {
		if _, malformedErr := rt.DecodeRecord(data, 2, 128); malformedErr == nil {
			t.Fatal("malformed record accepted")
		}
	}
	if _, limitErr := rt.DecodeRecord(encoded, 2, 1); !errors.Is(limitErr, rt.ErrTooLarge) {
		t.Fatal("decode limit ignored")
	}
	empty, err := rt.EncodeRecord(nil, 4)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rt.DecodeRecord(empty, 0, 4); err != nil {
		t.Fatal(err)
	}
}

func TestFloatSliceCodec(t *testing.T) {
	t.Parallel()
	values := []float64{0, math.Copysign(0, -1), math.Inf(1), math.NaN(), 1.5}
	data, err := rt.EncodeFloat64s(values, 40)
	if err != nil {
		t.Fatal(err)
	}
	for index, value := range values {
		if binary.LittleEndian.Uint64(data[index*8:]) != math.Float64bits(value) {
			t.Fatal("float bits changed")
		}
	}
	if _, err := rt.EncodeFloat64s(values, 39); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal("batch limit ignored")
	}
}

func FuzzDecodeRecord(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 42})
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) < 4 {
			return
		}
		count := binary.LittleEndian.Uint32(input)
		if count > 1024 {
			return
		}
		fields, err := rt.DecodeRecord(input, int(count), 65536)
		if err != nil {
			return
		}
		canonical := make([]rt.Field, len(fields))
		for index, field := range fields {
			canonical[index] = rt.Field{ID: uint32(index + 1), Data: field}
		} // #nosec G115 -- Field count is bounded above.
		encoded, err := rt.EncodeRecord(canonical, 65536)
		if err != nil || !bytes.Equal(input, encoded) {
			t.Fatalf("noncanonical record: %v", err)
		}
	})
}
