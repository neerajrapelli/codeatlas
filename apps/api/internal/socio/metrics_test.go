package socio

import "testing"

func TestPercentile(t *testing.T) {
	sorted := []float64{1, 2, 3, 4, 10}
	if got := percentile(sorted, 0.9); got != 10 {
		t.Fatalf("p90 = %v want 10", got)
	}
	if got := percentile(sorted, 0.5); got != 3 {
		t.Fatalf("p50 = %v want 3", got)
	}
	if got := percentile(nil, 0.5); got != 0 {
		t.Fatalf("empty percentile = %v want 0", got)
	}
}
