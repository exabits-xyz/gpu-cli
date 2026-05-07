package types

// DeviceAuthStart is returned by GET /authenticate/auth-access-code.
type DeviceAuthStart struct {
	State     string `json:"state"`
	ExpiresIn int    `json:"expires_in"`
}

// DeviceAuthToken is returned once browser authorization succeeds.
type DeviceAuthToken struct {
	Token string `json:"token"`
}
