package gogen_test

import (
	"bytes"
	"testing"

	"dario.cat/cocoon/internal/gen/gogen"
	"dario.cat/cocoon/internal/manifest"
)

func TestContractGeneration(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(apiManifest))
	if err != nil {
		t.Fatal(err)
	}
	contracts, err := gogen.Contracts(m)
	if err != nil || !bytes.Contains(contracts, []byte("reflect.New")) || !bytes.Contains(contracts, []byte("faulted.owner.Use")) {
		t.Fatalf("contracts: %v", err)
	}
	m.Resources = nil
	contracts, err = gogen.Contracts(m)
	if err != nil || bytes.Contains(contracts, []byte("reflect")) {
		t.Fatalf("stateless contracts: %v", err)
	}
	for _, helpers := range [][]string{nil, {"memory_init", "memory_copy", "memory_fill", "memory_zero"}} {
		if _, bulkErr := gogen.BulkTests(helpers); bulkErr != nil {
			t.Fatal(bulkErr)
		}
	}
	if _, err := gogen.BulkTests([]string{"unknown"}); err == nil {
		t.Fatal("unknown helper accepted")
	}
}
