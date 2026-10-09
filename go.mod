module dario.cat/cocoon

go 1.26.0

tool github.com/ncruces/wasm2go

require (
	github.com/pelletier/go-toml/v2 v2.4.3
	github.com/tetratelabs/wazero v1.12.0
	golang.org/x/mod v0.41.0
	golang.org/x/tools v0.50.0
)

require (
	github.com/ncruces/wasm2go v0.4.16 // indirect
	golang.org/x/sys v0.48.0 // indirect
)

// Published privately before Apache-2.0 licensing and dependency notices.
// Use v0.2.0 or later at dario.cat/cocoon.
retract v0.1.0
