package blastradius

import "testing"

func TestBfsDependents_shallowestDepth(t *testing.T) {
	// 2->1, 3->1, 4->2, 5->3 — target 1
	deps := []struct{ from, to int64 }{
		{2, 1}, {3, 1}, {4, 2}, {5, 3},
	}
	in := inboundAdjacency(deps)
	got := bfsDependents(1, in, 3)

	want := map[int64]int{2: 1, 3: 1, 4: 2, 5: 2}
	for id, d := range want {
		if got[id] != d {
			t.Fatalf("file %d depth: got %d want %d", id, got[id], d)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("got %d nodes want %d: %v", len(got), len(want), got)
	}
}

func TestBfsDependents_respectsDepthLimit(t *testing.T) {
	deps := []struct{ from, to int64 }{
		{2, 1}, {4, 2}, {6, 4},
	}
	in := inboundAdjacency(deps)
	got := bfsDependents(1, in, 2)
	if _, ok := got[6]; ok {
		t.Fatal("depth 3 node should be excluded at maxDepth=2")
	}
	if got[4] != 2 {
		t.Fatalf("file 4 depth: got %d want 2", got[4])
	}
}

func TestBfsDependents_emptyWhenNoInbound(t *testing.T) {
	got := bfsDependents(99, inboundAdjacency(nil), 5)
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}
