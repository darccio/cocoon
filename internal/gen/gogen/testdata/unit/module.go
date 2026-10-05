package unitfixture

import (
	"encoding/binary"

	"github.com/darccio/cocoon/rt"
)

// This authored typed-export mock deliberately permits replies a verified guest
// should not publish. The generated facade still owns every runtime boundary.
type fixtureReply struct {
	message          string
	pointer, size    uint32
	status           rt.Status
	shrinkDescriptor bool
}

type module struct {
	memory     []byte
	closeCalls map[uint64]int
	closeReply fixtureReply
	addReply   fixtureReply
	unitReply  fixtureReply
	nextHandle uint64
	addCalls   int
	unitCalls  int
}

func newModule(_ Options) (*module, *rt.Host) {
	return &module{
		memory:     make([]byte, 64, 128),
		closeCalls: make(map[uint64]int),
		closeReply: fixtureReply{pointer: 16},
		addReply:   fixtureReply{pointer: 16},
		unitReply:  fixtureReply{pointer: 16},
	}, nil
}

func (m *module) Memory() []byte        { return m.memory }
func (*module) Version() uint32         { return rt.ABIVersion }
func (*module) Schema() uint64          { return schemaHash }
func (*module) Init()                   {}
func (*module) Reserve(_ uint32) uint32 { return 32 }
func (*module) Output() uint32          { return 0 }
func (*module) Trim(_ uint32)           {}

func (m *module) publish(reply fixtureReply) int32 {
	binary.LittleEndian.PutUint32(m.memory, reply.pointer)
	binary.LittleEndian.PutUint32(m.memory[4:], reply.size)
	copy(m.memory[16:], reply.message)
	if reply.shrinkDescriptor {
		m.memory = m.memory[:4]
	}
	return int32(reply.status)
}

func (m *module) Xcocoon_item_new() int32 {
	m.nextHandle++
	binary.LittleEndian.PutUint64(m.memory[16:], uint64(1)<<32|m.nextHandle)
	return m.publish(fixtureReply{pointer: 16, size: 8})
}

func (m *module) Xcocoon_item_add(_ int64) int32 {
	m.addCalls++
	return m.publish(m.addReply)
}

func (m *module) Xcocoon_item_close(handle int64) int32 {
	m.closeCalls[uint64(handle)]++ // #nosec G115 -- Preserve the Wasm i64 handle's unsigned bits.
	return m.publish(m.closeReply)
}

func (m *module) Xcocoon_unit() int32 {
	m.unitCalls++
	return m.publish(m.unitReply)
}
