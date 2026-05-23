package tool

import (
	"context"
	"testing"
)

func TestRegistryRegisterAndList(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register(Spec{Name: "b_tool", RequiredScope: "b:read"}, func(ctx context.Context, req InvocationRequest) (InvocationResponse, error) {
		return InvocationResponse{Status: "success"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = reg.Register(Spec{Name: "a_tool", RequiredScope: "a:read"}, func(ctx context.Context, req InvocationRequest) (InvocationResponse, error) {
		return InvocationResponse{Status: "success"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	specs := reg.List()
	if len(specs) != 2 || specs[0].Name != "a_tool" {
		t.Fatalf("expected sorted specs, got %+v", specs)
	}
	reg.Unregister("a_tool")
	if _, ok := reg.Get("a_tool"); ok {
		t.Fatal("expected a_tool to be unregistered")
	}
}
