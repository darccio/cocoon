package batchfixture_test

import (
	"errors"
	"reflect"
	"testing"

	"dario.cat/cocoon/rt"
	fixture "example.com/batch-fixture"
)

// Embedding an interface promotes its unexported marker across package boundaries.
// Only the generator's concrete implementation types may become executable plans.
type foreignCommand struct{ fixture.StreamCommand }

type foreignIdleCommand struct {
	fixture.IdleCommand
	fixture.StreamCommand
}

func TestExternalIdleRejectsAnotherResourceCommand(t *testing.T) {
	t.Parallel()
	library, err := fixture.Open(fixture.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := library.Close(); err != nil {
			t.Error(err)
		}
	})
	idle, err := library.NewIdle()
	if err != nil {
		t.Fatal(err)
	}
	streamCommand := fixture.StreamUnitCommand()
	if _, sharedMarker := any(streamCommand).(fixture.IdleCommand); sharedMarker {
		t.Fatal("resource command markers are interchangeable")
	}
	// A foreign embedding can satisfy Idle's marker even though its payload is
	// a Stream command. Exact dynamic-type preflight must still reject it.
	results, err := idle.Batch(t.Context(), []fixture.IdleCommand{foreignIdleCommand{StreamCommand: streamCommand}}, rt.BatchLimits{})
	var failure *rt.BatchError
	var argument *rt.AppError
	if len(results) != 0 || !errors.As(err, &failure) || failure.Index != 0 || failure.Attempted || !errors.As(err, &argument) || argument.Status != rt.ErrArg {
		t.Fatal("cross-resource embedding reached Idle", results, err)
	}
	results, err = idle.Batch(t.Context(), nil, rt.BatchLimits{})
	if err != nil || len(results) != 0 {
		t.Fatal("rejected foreign command damaged idle ownership", err)
	}
}

func TestExternalConsumerRejectsForeignDynamicCommandTypes(t *testing.T) {
	t.Parallel()
	library, err := fixture.Open(fixture.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := library.Close(); err != nil {
			t.Error(err)
		}
	})
	stream, err := library.NewStream()
	if err != nil {
		t.Fatal(err)
	}
	valid := fixture.StreamUnitCommand()
	pointer := reflect.New(reflect.TypeOf(valid))
	pointer.Elem().Set(reflect.ValueOf(valid))
	pointed, ok := pointer.Interface().(fixture.StreamCommand)
	if !ok {
		t.Fatal("generated value marker no longer promotes to pointer")
	}
	nilPointer, ok := reflect.Zero(pointer.Type()).Interface().(fixture.StreamCommand)
	if !ok {
		t.Fatal("typed nil lost generated command marker")
	}
	for _, command := range []fixture.StreamCommand{nil, foreignCommand{StreamCommand: valid}, &foreignCommand{StreamCommand: valid}, (*foreignCommand)(nil), pointed, nilPointer} {
		results, err := stream.Batch(t.Context(), []fixture.StreamCommand{valid, command, valid}, rt.BatchLimits{})
		var failure *rt.BatchError
		var argument *rt.AppError
		if len(results) != 0 || !errors.As(err, &failure) || failure.Index != 1 || failure.Attempted || !errors.As(err, &argument) || argument.Status != rt.ErrArg {
			t.Fatalf("foreign %T bypassed exact type rejection: %v %v", command, results, err)
		}
	}
	// A reflected copy keeps the exact concrete dynamic type and must work.
	reflected, ok := reflect.ValueOf(valid).Interface().(fixture.StreamCommand)
	if !ok {
		t.Fatal("reflected exact value changed marker")
	}
	results, err := stream.Batch(t.Context(), []fixture.StreamCommand{reflected}, rt.BatchLimits{})
	if err != nil || len(results) != 1 {
		t.Fatal(results, err)
	}
	if _, ok := results[0].(fixture.StreamUnitResult); !ok {
		t.Fatalf("external result tag is %T", results[0])
	}
}

func TestExternalConsumerNilContextAndResource(t *testing.T) {
	t.Parallel()
	library, err := fixture.Open(fixture.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := library.Close(); err != nil {
			t.Error(err)
		}
	})
	stream, err := library.NewStream()
	if err != nil {
		t.Fatal(err)
	}
	results, err := stream.Batch(nil, nil, rt.BatchLimits{})
	var failure *rt.BatchError
	var argument *rt.AppError
	if len(results) != 0 || !errors.As(err, &failure) || failure.Index != -1 || failure.Attempted || !errors.As(err, &argument) || argument.Status != rt.ErrArg {
		t.Fatal("nil context panicked or changed classification", err)
	}
	var missing *fixture.Stream
	results, err = missing.Batch(t.Context(), nil, rt.BatchLimits{})
	var invalid *rt.HandleError
	if len(results) != 0 || !errors.As(err, &failure) || failure.Index != -1 || failure.Attempted || !errors.As(err, &invalid) {
		t.Fatal("nil resource panicked or changed classification", err)
	}
}
