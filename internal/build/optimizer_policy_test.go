package build

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
)

type optimizerPolicyCall struct {
	directory, program string
	args               []string
}

type optimizerPolicyRunner struct {
	calls  []optimizerPolicyCall
	runner pipelineRunner
}

func (r *optimizerPolicyRunner) Run(ctx context.Context, directory, program string, args ...string) ([]byte, error) {
	record := program == "wasm-metadce" || program == "wasm-opt" ||
		program == "go" && len(args) >= 2 && args[0] == "tool" && args[1] == "wasm2go"
	if record && !slices.Contains(args, "--version") && !slices.Contains(args, "-version") {
		r.calls = append(r.calls, optimizerPolicyCall{args: slices.Clone(args), directory: directory, program: program})
	}
	return r.runner.Run(ctx, directory, program, args...)
}

func TestPipelineOptimizerPolicy(t *testing.T) {
	t.Parallel()
	directory, runner := makeProject(t)
	recording := &optimizerPolicyRunner{runner: runner}
	p := pipeline{runner: recording, sources: fakeSources}
	if _, err := p.run(t.Context(), directory, testManifest(t)); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(directory, ".cocoon-build")
	pruned, final := filepath.Join(work, "pruned.wasm"), filepath.Join(work, "final.wasm")
	// Literal expectations also guard against feature changes and unsafe flags.
	features := []string{
		"--mvp-features", "--enable-mutable-globals", "--enable-bulk-memory",
		"--enable-sign-ext", "--enable-nontrapping-float-to-int",
		"--enable-reference-types", "--enable-multivalue",
	}
	want := []optimizerPolicyCall{
		{
			args: append([]string{
				filepath.Join(work, "target", "wasm32-unknown-unknown", "release", "test_shim.wasm"),
				"--graph-file", filepath.Join(work, "exports.json"), "-o", pruned,
			}, features...),
			directory: directory,
			program:   "wasm-metadce",
		},
		{
			args:      append([]string{pruned, "-O3", "-o", final}, features...),
			directory: directory,
			program:   "wasm-opt",
		},
		{
			args:      []string{"tool", "wasm2go", "-unsafe", "-pkg", "wasm", "-o", filepath.Join(work, "translated.go"), final},
			directory: directory,
			program:   "go",
		},
	}
	if len(recording.calls) != len(want) {
		t.Fatalf("optimizer pipeline calls = %v, want %v", recording.calls, want)
	}
	for index, expected := range want {
		actual := recording.calls[index]
		if actual.directory != expected.directory || actual.program != expected.program || !slices.Equal(actual.args, expected.args) {
			t.Fatalf("optimizer pipeline call %d = %v, want %v", index, actual, expected)
		}
	}
}
