package ai

import "context"

type ValidateMentionsRequest struct {
	Paths     []string `json:"paths"`
	RuleNames []string `json:"ruleNames"`
}

type ValidateMentionsResponse struct {
	Paths map[string]bool `json:"paths"`
	Rules map[string]bool `json:"rules"`
}

func (s *Service) ValidateMentions(ctx context.Context, repositoryID int64, tenantID string, req ValidateMentionsRequest) (ValidateMentionsResponse, error) {
	pathOK, ruleOK, err := s.retriever.ValidateMentions(ctx, repositoryID, tenantID, req.Paths, req.RuleNames)
	if err != nil {
		return ValidateMentionsResponse{}, err
	}
	return ValidateMentionsResponse{Paths: pathOK, Rules: ruleOK}, nil
}

func (s *Service) GuardStreamAnswer(ctx context.Context, repositoryID int64, tenantID, answer string) (string, ValidateMentionsResponse, error) {
	paths := ExtractPathMentions(answer)
	rules := ExtractRuleMentions(answer)
	pathOK, ruleOK, err := s.retriever.ValidateMentions(ctx, repositoryID, tenantID, paths, rules)
	if err != nil {
		return answer, ValidateMentionsResponse{}, err
	}
	sanitized := SanitizeAnswer(answer, pathOK, ruleOK)
	return sanitized, ValidateMentionsResponse{Paths: pathOK, Rules: ruleOK}, nil
}
