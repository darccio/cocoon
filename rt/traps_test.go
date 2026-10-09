package rt_test

import (
	"errors"
	"testing"

	"dario.cat/cocoon/rt"
	"dario.cat/cocoon/rt/internal/trapfix"
)

type trapModule struct{ *trapfix.Module }

func (m *trapModule) Memory() []byte        { return *m.Xmemory().Slice() }
func (*trapModule) Version() uint32         { return 3 }
func (*trapModule) Schema() uint64          { return 7 }
func (*trapModule) Init()                   {}
func (*trapModule) Reserve(_ uint32) uint32 { return 32 }
func (*trapModule) Output() uint32          { return 0 }
func (*trapModule) Trim(_ uint32)           {}

func TestPinnedTranslatorTrapContract(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		call func(*trapfix.Module)
		name string
		kind rt.FaultKind
	}{
		{func(m *trapfix.Module) { m.Xunreachable() }, "unreachable", rt.FaultUnreachable},
		{func(m *trapfix.Module) { m.Xload_oob() }, "memory", rt.FaultMemory},
		{func(m *trapfix.Module) { m.Xdiv_zero() }, "divide zero", rt.FaultArithmetic},
		{func(m *trapfix.Module) { m.Xi32_overflow() }, "i32 overflow", rt.FaultArithmetic},
		{func(m *trapfix.Module) { m.Xi64_overflow() }, "i64 overflow", rt.FaultArithmetic},
		{func(m *trapfix.Module) { m.Xbad_conversion() }, "conversion", rt.FaultArithmetic},
		{func(m *trapfix.Module) { m.Xindirect_null() }, "null", rt.FaultIndirectCall},
		{func(m *trapfix.Module) { m.Xindirect_mismatch() }, "signature", rt.FaultIndirectCall},
		{func(m *trapfix.Module) { m.Xtable_oob() }, "table call", rt.FaultTable},
		{func(m *trapfix.Module) { m.Xtable_get_oob() }, "table get", rt.FaultTable},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			m := &trapModule{Module: trapfix.New()}
			instance, err := rt.NewInstance(m, 7, rt.Limits{MaxInput: 64, MaxOutput: 64, MaxMemory: 3 * 65536})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(instance.Close)
			err = instance.Call(test.name, func(_ *rt.Call) error { test.call(m.Module); return nil })
			var fault *rt.FaultError
			if !errors.As(err, &fault) || fault.Kind != test.kind || instance.Healthy() {
				t.Fatalf("trap: %v; want %v", err, test.kind)
			}
		})
	}
}

func TestPinnedBulkMemorySpareCapacity(t *testing.T) {
	t.Parallel()
	m := trapfix.New()
	if m.Xgrow(1) != 1 || m.Xgrow(1) != 2 || m.Xgrow(1) != -1 {
		t.Fatal("memory growth limits")
	}
	memory := m.Xmemory().Slice()
	bounded := make([]byte, len(*memory), len(*memory)+65536)
	copy(bounded, *memory)
	*memory = bounded
	end := int32(len(bounded)) // #nosec G115 -- The fixture memory maximum is three pages.
	for _, invoke := range []func(){
		func() { m.Xfill(end, 1, 1) }, func() { m.Xfill(end+1, 1, 0) },
		func() { m.Xzero(end, 1) }, func() { m.Xzero(end+1, 0) },
		func() { m.Xcopy(end, 0, 1) }, func() { m.Xcopy(0, end, 1) },
		func() { m.Xinit(end, 0, 1) }, func() { m.Xinit(0, 4, 0) },
	} {
		func() {
			defer func() {
				value := recover()
				if value == nil {
					t.Error("bulk operation crossed logical memory or data length")
				}
			}()
			invoke()
		}()
	}
	m.Xinit(32, 0, 3)
	m.Xcopy(33, 32, 3)
	if string((*memory)[32:36]) != "aabc" {
		t.Fatal("overlapping copy is not memmove")
	}
	m.Xzero(end, 0)
}
