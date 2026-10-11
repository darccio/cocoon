package harden

// These shapes describe translator support code, not a payload implementation.
// Eligibility is optional: any drift leaves the scalar-view pass inactive.
const memoryViewSupport = `package wasm
const big = false ||
 runtime.GOARCH == "ppc64" || runtime.GOARCH == "s390x" ||
 runtime.GOARCH == "mips" || runtime.GOARCH == "mips64"
const unalignedOK = false ||
 runtime.GOARCH == "386" || runtime.GOARCH == "amd64" ||
 runtime.GOARCH == "arm64" || runtime.GOARCH == "loong64" ||
 runtime.GOARCH == "ppc64" || runtime.GOARCH == "ppc64le" ||
 runtime.GOARCH == "s390x" || runtime.GOARCH == "wasm"
func i32(x int32) int32 { return x }
func memory_grow(mem *[]byte, delta, max int64) int64 {
 buf := *mem
 len := len(buf)
 old := len >> 16
 if delta == 0 { return int64(old) }
 max = int64(min(uint64(max), math.MaxInt>>16))
 new, c := bits.Add64(uint64(old), uint64(delta), 0)
 if c != 0 || new > uint64(max) { return -1 }
 *mem = append(buf, make([]byte, int(new<<16)-len)...)
 return int64(old)
}
`

const memoryViewLoad = `
func load%[1]d[T uint32 | uint64](mem []byte, addr T) uint%[1]d {
 if !unalignedOK { return binary.LittleEndian.Uint%[1]d(mem[addr:]) }
 _ = mem[uint64(addr)+%[2]d]
 val := *(*uint%[1]d)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr)))
 if big { return bits.ReverseBytes%[1]d(val) }
 return val
}
`

const memoryViewStore = `
func store%[1]d[T uint32 | uint64](mem []byte, addr T, val uint%[1]d) {
 if !unalignedOK { binary.LittleEndian.PutUint%[1]d(mem[addr:], val); return }
 if big { val = bits.ReverseBytes%[1]d(val) }
 _ = mem[uint64(addr)+%[2]d]
 *(*uint%[1]d)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr))) = val
}
`
