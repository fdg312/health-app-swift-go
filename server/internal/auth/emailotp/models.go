package emailotp

type RequestResponse struct {
	Status    string  `json:"status"`
	DebugCode *string `json:"debug_code,omitempty"`
}

type VerifyResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	UserID      string `json:"user_id"`
}
