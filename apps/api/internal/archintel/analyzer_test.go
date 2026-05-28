package archintel

import "testing"

func TestAnalyzeDiscussionExtractsDecisionCandidates(t *testing.T) {
	a := NewAnalyzer(false)
	items := a.AnalyzeDiscussion(DiscussionInput{
		SourceKind: "pr_review",
		Title:      "Architecture decision: queue-based batching",
		Body:       "We accepted and decided to batch ingest workers for performance and complexity tradeoff in apps/api/internal/ingestion/service.go.",
		Author:     "maintainerA",
	})
	if len(items) == 0 {
		t.Fatalf("expected at least one decision candidate")
	}
	if items[0].Status != DecisionAccepted {
		t.Fatalf("expected accepted status, got %s", items[0].Status)
	}
	if len(items[0].Participants) != 1 || items[0].Participants[0] != "maintainerA" {
		t.Fatalf("expected participant to be preserved")
	}
}

func TestAnalyzeDiscussionRejectStatusInference(t *testing.T) {
	a := NewAnalyzer(false)
	items := a.AnalyzeDiscussion(DiscussionInput{
		Title: "RFC: cache layer redesign",
		Body:  "Architecture decision proposal rejected due to complexity and cost concerns.",
	})
	if len(items) == 0 {
		t.Fatalf("expected at least one candidate")
	}
	if items[0].Status != DecisionRejected {
		t.Fatalf("expected rejected status, got %s", items[0].Status)
	}
}
