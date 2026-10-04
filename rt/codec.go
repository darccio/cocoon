package rt

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Field is one numbered, length-delimited ABI v3 record field.
type Field struct {
	Data []byte
	ID   uint32
}

// RecordSize checks the complete record before encoding allocates memory.
func RecordSize(fields []Field, limit uint64) (uint64, error) {
	size, err := InputSize(limit, uint64(len(fields)), 8, 4)
	if err != nil {
		return 0, err
	}
	for index, field := range fields {
		if uint64(field.ID) != uint64(index)+1 {
			return 0, fmt.Errorf("%w: record field order", ErrProtocol)
		}
		size, err = InputSize(limit, 1, size, uint64(len(field.Data)))
		if err != nil {
			return 0, err
		}
	}
	return size, nil
}

// EncodeRecord encodes required fields in canonical order with explicit lengths.
func EncodeRecord(fields []Field, limit uint64) ([]byte, error) {
	size, err := RecordSize(fields, limit)
	if err != nil {
		return nil, err
	}
	output := make([]byte, 0, int(size))                                   // #nosec G115 -- RecordSize checks the platform int bound.
	output = binary.LittleEndian.AppendUint32(output, uint32(len(fields))) // #nosec G115 -- RecordSize bounds field count.
	for _, field := range fields {
		output = binary.LittleEndian.AppendUint32(output, field.ID)
		output = binary.LittleEndian.AppendUint32(output, uint32(len(field.Data))) // #nosec G115 -- RecordSize bounds each field length.
		output = append(output, field.Data...)
	}
	return output, nil
}

// DecodeRecord rejects missing, duplicate, unknown, unordered, or trailing fields.
// Returned field views borrow data; callers must respect its lifetime.
func DecodeRecord(data []byte, expected int, limit uint64) ([][]byte, error) {
	if uint64(len(data)) > limit {
		return nil, ErrTooLarge
	}
	if expected < 0 || expected > (len(data)-4)/8 || len(data) < 4 ||
		uint64(binary.LittleEndian.Uint32(data)) != uint64(expected) {
		return nil, fmt.Errorf("%w: record field count", ErrProtocol)
	}
	fields := make([][]byte, expected)
	remaining := data[4:]
	for index := range expected {
		if len(remaining) < 8 || uint64(binary.LittleEndian.Uint32(remaining)) != uint64(index)+1 {
			return nil, fmt.Errorf("%w: record field order", ErrProtocol)
		}
		size := uint64(binary.LittleEndian.Uint32(remaining[4:]))
		remaining = remaining[8:]
		if size > uint64(len(remaining)) {
			return nil, fmt.Errorf("%w: truncated record field", ErrProtocol)
		}
		fields[index] = remaining[:size:size]
		remaining = remaining[size:]
	}
	if len(remaining) != 0 {
		return nil, fmt.Errorf("%w: trailing record data", ErrProtocol)
	}
	return fields, nil
}

// EncodeFloat64s writes portable little-endian elements after checking their size.
func EncodeFloat64s(values []float64, limit uint64) ([]byte, error) {
	size, err := InputSize(limit, uint64(len(values)), 8)
	if err != nil {
		return nil, err
	}
	data := make([]byte, 0, int(size)) // #nosec G115 -- InputSize checks the platform int bound.
	for _, value := range values {
		data = binary.LittleEndian.AppendUint64(data, math.Float64bits(value))
	}
	return data, nil
}
