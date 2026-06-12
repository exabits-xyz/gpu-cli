package types

// ModelProvider describes the organization that serves an AI model.
type ModelProvider struct {
	Name              string `json:"name"`
	Description       string `json:"description,omitempty"`
	PrivacyPolicyURL  string `json:"privacy_policy_url,omitempty"`
	TermsOfServiceURL string `json:"terms_of_service_url,omitempty"`
	FaviconURL        string `json:"favicon_url,omitempty"`
	Headquarters      string `json:"headquarters,omitempty"`
}

// Model represents an AI model as returned by GET /models.
// Prices are per million tokens, keyed by currency (e.g. "usd").
type Model struct {
	ID                  string             `json:"id"`
	ModelName           string             `json:"model_name"`
	DisplayName         string             `json:"display_name"`
	Description         string             `json:"description,omitempty"`
	Provider            ModelProvider      `json:"provider"`
	HFRepo              string             `json:"hf_repo,omitempty"`
	InputTokensPrice    map[string]float64 `json:"input_tokens_price"`
	OutputTokensPrice   map[string]float64 `json:"output_tokens_price"`
	ContextLength       int                `json:"context_length"`
	MaxCompletionTokens int                `json:"max_completion_tokens"`
	CanonicalSlug       string             `json:"canonical_slug"`
	KnowledgeCutoff     string             `json:"knowledge_cutoff,omitempty"` // ISO 8601 timestamp
}

// ModelListResult is the output shape for `model list`.
// Total reflects the server-side record count before limit/offset are applied.
type ModelListResult struct {
	Total int     `json:"total"`
	Data  []Model `json:"data"`
}
