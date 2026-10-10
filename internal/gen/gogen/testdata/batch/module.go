package batchfixture

import (
	"bytes"
	"encoding/binary"
	"math"

	"dario.cat/cocoon/rt"
)

// Authored export mock: replies can intentionally violate the guest contract,
// while every execution/encoding/ownership boundary remains generated code.
type fixtureReply struct {
	data      []byte
	status    rt.Status
	afterSize int
	override  bool
	grow      bool
}

type mixedInputs struct {
	buffers    map[string][]byte
	signed32   int32
	unsigned32 uint32
	signed64   int64
	unsigned64 uint64
	single     float32
	double     float64
	flag       int32
}

type module struct {
	memory       []byte
	replies      []fixtureReply
	operations   []string
	handles      []uint64
	mixed        mixedInputs
	cancel       func()
	started      chan<- struct{}
	release      <-chan struct{}
	blockAt      int
	cancelAt     int
	reserveCalls int
	closeCalls   int
	newCalls     int
	nextHandle   uint64
}

func newModule(_ Options) (*module, *rt.Host) {
	return &module{memory: make([]byte, 65536, 131072)}, nil
}

func (m *module) Memory() []byte          { return m.memory }
func (*module) Version() uint32           { return rt.ABIVersion }
func (*module) Schema() uint64            { return schemaHash }
func (*module) Init()                     {}
func (m *module) Reserve(_ uint32) uint32 { m.reserveCalls++; return 32 }
func (*module) Output() uint32            { return 0 }
func (*module) Trim(_ uint32)             {}

func (m *module) publish(data []byte, status rt.Status) int32 {
	copy(m.memory[16384:], data)
	binary.LittleEndian.PutUint32(m.memory, 16384)
	binary.LittleEndian.PutUint32(m.memory[4:], uint32(len(data)))
	return int32(status)
}

func (m *module) finish(op string, handle int64, data []byte) int32 {
	m.operations = append(m.operations, op)
	m.handles = append(m.handles, uint64(handle))
	index := len(m.operations)
	if index == m.blockAt {
		close(m.started)
		<-m.release
	}
	if index == m.cancelAt {
		m.cancel()
	}
	var reply fixtureReply
	if index <= len(m.replies) {
		reply = m.replies[index-1]
	}
	if reply.override {
		data = reply.data
	}
	if reply.grow {
		m.memory = append(bytes.Clone(m.memory), make([]byte, 65536)...)
	}
	status := m.publish(data, reply.status)
	if reply.afterSize != 0 {
		m.memory = append(m.memory, make([]byte, reply.afterSize-len(m.memory))...)
	}
	return status
}

func (m *module) input(pointer, size int32) []byte {
	return bytes.Clone(m.memory[uint32(pointer) : uint32(pointer)+uint32(size)])
}

func (m *module) Xcocoon_stream_new() int32 {
	m.newCalls++
	m.nextHandle++
	return m.publish(binary.LittleEndian.AppendUint64(nil, 0xfedcba98_00000000|m.nextHandle), rt.OK)
}
func (m *module) Xcocoon_stream_close(_ int64) int32 {
	m.closeCalls++
	return m.publish(nil, rt.OK)
}

func (m *module) Xcocoon_idle_new() int32                 { return m.Xcocoon_stream_new() }
func (m *module) Xcocoon_idle_close(handle int64) int32   { return m.Xcocoon_stream_close(handle) }
func (m *module) Xcocoon_scalar_new() int32               { return m.Xcocoon_stream_new() }
func (m *module) Xcocoon_scalar_close(handle int64) int32 { return m.Xcocoon_stream_close(handle) }
func (m *module) Xcocoon_scalar_count(handle, value int64) int32 {
	return m.finish("scalar_count", handle, binary.LittleEndian.AppendUint64(nil, uint64(value)))
}
func (m *module) Xcocoon_scalar_unit(handle int64) int32 { return m.finish("scalar_unit", handle, nil) }
func (m *module) Xcocoon_stream_echo(handle int64, pointer, size int32) int32 {
	return m.finish("echo", handle, m.input(pointer, size))
}
func (m *module) Xcocoon_stream_text(handle int64, pointer, size int32) int32 {
	return m.finish("text", handle, m.input(pointer, size))
}
func (m *module) Xcocoon_stream_count(handle, value int64) int32 {
	return m.finish("count", handle, binary.LittleEndian.AppendUint64(nil, uint64(value)))
}
func (m *module) Xcocoon_stream_flag(handle int64, value int32) int32 {
	return m.finish("flag", handle, []byte{byte(value)})
}
func (m *module) Xcocoon_stream_float(handle int64, value float64) int32 {
	return m.finish("float", handle, binary.LittleEndian.AppendUint64(nil, math.Float64bits(value)))
}
func (m *module) Xcocoon_stream_numbers(handle int64, pointer, size int32) int32 {
	return m.finish("numbers", handle, m.input(pointer, size))
}
func (m *module) Xcocoon_stream_unit(handle int64) int32 { return m.finish("unit", handle, nil) }

func (m *module) Xcocoon_stream_mix(handle int64, signed32, unsigned32 int32, signed64, unsigned64 int64, single float32, double float64, flag int32, textPointer, textSize, dataPointer, dataSize, signed32sPointer, signed32sSize, unsigned32sPointer, unsigned32sSize, signed64sPointer, signed64sSize, unsigned64sPointer, unsigned64sSize, singlesPointer, singlesSize, doublesPointer, doublesSize, envelopePointer, envelopeSize int32) int32 {
	m.mixed = mixedInputs{
		signed32: signed32, unsigned32: uint32(unsigned32), signed64: signed64, unsigned64: uint64(unsigned64), single: single, double: double, flag: flag,
		buffers: map[string][]byte{
			"text": m.input(textPointer, textSize), "data": m.input(dataPointer, dataSize),
			"signed32s": m.input(signed32sPointer, signed32sSize), "unsigned32s": m.input(unsigned32sPointer, unsigned32sSize),
			"signed64s": m.input(signed64sPointer, signed64sSize), "unsigned64s": m.input(unsigned64sPointer, unsigned64sSize),
			"singles": m.input(singlesPointer, singlesSize), "doubles": m.input(doublesPointer, doublesSize),
			"envelope": m.input(envelopePointer, envelopeSize),
		},
	}
	return m.finish("mix", handle, m.mixed.buffers["envelope"])
}
