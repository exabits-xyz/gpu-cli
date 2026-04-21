package types

// CreditBalance is the data object returned by GET /billing/balance.
// The API returns available credits keyed by currency (e.g. "USD").
type CreditBalance struct {
	Available map[string]float64 `json:"available"`
}

// BillingResource is the resource info embedded in a UsageRecord.
type BillingResource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"` // "vm" or "volume"
}

// UsageRecord is a single entry in the resource usage history.
type UsageRecord struct {
	ID             string          `json:"id"`
	Resource       BillingResource `json:"resource"`
	CreatedTime    int64           `json:"created_time"`              // Unix timestamp
	Status         string          `json:"status"`                    // "active" | "terminated"
	TerminatedTime int64           `json:"terminated_time,omitempty"` // Unix timestamp; 0 if still active
	TotalUptime    int64           `json:"total_uptime"`              // minutes
	TotalFee       float64         `json:"total_fee"`
	PerMinuteFee   float64         `json:"per_minute_fee"`
}

// UsageListResult is the output shape for `billing usage`.
type UsageListResult struct {
	Total int           `json:"total"`
	Data  []UsageRecord `json:"data"`
}

// Statement is a single billing statement entry.
type Statement struct {
	ID           string  `json:"id"`
	ResourceID   string  `json:"resource_id"`
	ResourceType string  `json:"resource_type"` // "vm" | "volume"
	Status       string  `json:"status"`        // "paid", etc.
	StartedTime  int64   `json:"started_time"`  // Unix timestamp
	DueTime      int64   `json:"due_time"`      // Unix timestamp
	Type         string  `json:"type"`          // "lease_fee", etc.
	Amount       float64 `json:"amount"`
}

// StatementListResult is the output shape for `billing statement`.
type StatementListResult struct {
	Total int         `json:"total"`
	Data  []Statement `json:"data"`
}
