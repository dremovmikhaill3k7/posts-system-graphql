package graph

import "testing"

func TestLimitOffset(t *testing.T) {
	limit := 50
	offset := 3
	l, o := limitOffset(&limit, &offset)
	if l != 50 || o != 3 {
		t.Fatalf("got limit=%d offset=%d", l, o)
	}

	huge := 500
	l, o = limitOffset(&huge, nil)
	if l != maxLimit || o != defaultOffset {
		t.Fatalf("expected capped limit %d, got %d", maxLimit, l)
	}
}
