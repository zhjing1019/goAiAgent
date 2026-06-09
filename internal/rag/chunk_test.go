package rag

import "testing"

func TestSplitText(t *testing.T) {
	short := SplitText("hello", 10)
	if len(short) != 1 || short[0] != "hello" {
		t.Fatalf("unexpected: %v", short)
	}

	long := SplitText("一二三四五六七八九十", 3)
	if len(long) != 4 {
		t.Fatalf("expected 4 chunks, got %d: %v", len(long), long)
	}
}
