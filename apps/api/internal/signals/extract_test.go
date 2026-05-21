package signals

import "testing"

func TestExtractFromTextDetectsDebtAndPaths(t *testing.T) {
	text := "We have tech debt in src/api/handler.go; TODO: refactor auth module"
	drafts := ExtractFromText(text)
	if len(drafts) == 0 {
		t.Fatal("expected at least one signal")
	}
	found := false
	for _, d := range drafts {
		if d.SignalType == TypeTechnicalDebt {
			found = true
			if len(d.FilePaths) == 0 {
				t.Error("expected file path extraction")
			}
		}
	}
	if !found {
		t.Fatalf("expected technical_debt, got %+v", drafts)
	}
}

func TestExtractArchitecturalDecision(t *testing.T) {
	drafts := ExtractFromText("ADR: we decided to split the monolith boundary")
	for _, d := range drafts {
		if d.SignalType == TypeArchitecturalDecision {
			return
		}
	}
	t.Fatal("expected architectural_decision")
}
