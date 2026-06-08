package agent

import (
	"context"
	"testing"
)

// 测试工具执行
func TestRegistryExecute(t *testing.T) {
	r := DefaultRegistry()

	result, err := r.Execute(context.Background(), "add_numbers", `{"a":3,"b":5}`)
	if err != nil {
		t.Fatal(err)
	}
	if result != `{"result": 8}` {
		t.Fatalf("unexpected result: %s", result)
	}

	result, err = r.Execute(context.Background(), "multiply_numbers", `{"a":6,"b":7}`)
	if err != nil {
		t.Fatal(err)
	}
	if result != `{"result": 42}` {
		t.Fatalf("unexpected multiply result: %s", result)
	}
}
// 测试未知工具执行
func TestRegistryUnknownTool(t *testing.T) {
	r := NewRegistry()
	_, err := r.Execute(context.Background(), "not_exist", `{}`)
	if err == nil {
		t.Fatal("expected error")
	}
}
// 测试获取当前时间工具
func TestGetCurrentTimeTool(t *testing.T) {
	var tool GetCurrentTimeTool
	out, err := tool.Execute(context.Background(), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 10 {
		t.Fatalf("unexpected output: %s", out)
	}
}
