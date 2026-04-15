package types

// CreateTokenRequest is the body for POST /api-tokens.
type CreateTokenRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// APIToken represents an API token as returned by create and list endpoints.
type APIToken struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Token       string `json:"token"`      // The token value used for authentication
	CreatedAt   int64  `json:"created_at"` // Unix timestamp
	LastUsed    int64  `json:"last_used"`  // Unix timestamp; 0 means never used
}

// TokenListResult is the output shape for `token list`.
// Total reflects the server-side record count before limit/offset are applied.
type TokenListResult struct {
	Total int        `json:"total"`
	Data  []APIToken `json:"data"`
}
