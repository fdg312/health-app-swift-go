package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const resendAPIURL = "https://api.resend.com/emails"

type ResendConfig struct {
	APIKey string
	From   string
}

type ResendSender struct {
	cfg    ResendConfig
	client *http.Client
}

func NewResendSender(cfg ResendConfig) *ResendSender {
	return &ResendSender{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
}

type resendResponse struct {
	ID string `json:"id"`
}

type resendErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Name       string `json:"name"`
	Message    string `json:"message"`
}

func (r *ResendSender) Send(to, subject, textBody string) error {
	payload := resendRequest{
		From:    r.cfg.From,
		To:      []string{to},
		Subject: subject,
		Text:    textBody,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("resend: failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, resendAPIURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("resend: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("resend: failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp resendErrorResponse
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil && errResp.Message != "" {
			return fmt.Errorf("resend: API error %d %s: %s", resp.StatusCode, errResp.Name, errResp.Message)
		}
		return fmt.Errorf("resend: API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
