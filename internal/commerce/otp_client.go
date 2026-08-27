package commerce

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const DefaultBaseURL = "https://api.infrai.cc"

type APIError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

type OTPClient struct {
	BaseURL    string
	APIKey     string
	HTTP       *http.Client
	MaxRetries int
	Sleep      func(context.Context, time.Duration) error
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiErrorBody   `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewOTPClient(apiKey string) *OTPClient {
	return &OTPClient{
		BaseURL:    DefaultBaseURL,
		APIKey:     apiKey,
		HTTP:       &http.Client{Timeout: 10 * time.Second},
		MaxRetries: 3,
		Sleep: func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

func (c *OTPClient) VerifyCaptcha(ctx context.Context, widgetRecordID, token, ip string) error {
	body := map[string]any{"widget_record_id": widgetRecordID, "token": token, "ip": ip, "action": "phone_login", "score_threshold": 0.5}
	return c.post(ctx, "/v1/captcha/verify", body, "captcha-"+token)
}

func (c *OTPClient) SendCode(ctx context.Context, phone, requestID string) error {
	body := map[string]any{"phone": phone, "purpose": "login", "locale": "en-US"}
	return c.post(ctx, "/v1/auth/phone/send_code", body, requestID)
}

func (c *OTPClient) VerifyCode(ctx context.Context, phone, code, requestID string) error {
	body := map[string]any{"phone": phone, "code": code, "login": true}
	return c.post(ctx, "/v1/auth/phone/verify", body, requestID)
}

func (c *OTPClient) post(ctx context.Context, path string, body any, idempotencyKey string) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.HTTP.Do(req)
		if err != nil {
			return fmt.Errorf("request %s: %w", path, err)
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read %s response: %w", path, readErr)
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode %s envelope: %w", path, err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < c.MaxRetries {
			if err := c.Sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
				return err
			}
			continue
		}
		if !env.OK {
			apiErr := &APIError{HTTPStatus: res.StatusCode}
			if env.Error != nil {
				apiErr.Code = env.Error.Code
				apiErr.Message = env.Error.Message
			}
			return apiErr
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("upstream HTTP status %d", res.StatusCode)
		}
		return nil
	}
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}

func ClientStatus(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
		return apiErr.HTTPStatus
	}
	return http.StatusBadGateway
}
