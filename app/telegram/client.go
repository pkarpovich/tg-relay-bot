package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.telegram.org"

type Config struct {
	Token      string
	HTTPClient *http.Client
	BaseURL    string
}

type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

func NewClient(cfg Config) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Client{
		token:      cfg.Token,
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	Parameters  *struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

type APIError struct {
	Code        int
	Description string
	RetryAfter  int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram api error (code %d): %s", e.Code, e.Description)
}

func (c *Client) do(ctx context.Context, method string, payload any) (json.RawMessage, error) {
	result, err := c.doOnce(ctx, method, payload)

	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusTooManyRequests && apiErr.RetryAfter > 0 {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait before %s retry: %w", method, ctx.Err())
		case <-time.After(time.Duration(apiErr.RetryAfter) * time.Second):
		}

		return c.doOnce(ctx, method, payload)
	}

	return result, err
}

func (c *Client) doOnce(ctx context.Context, method string, payload any) (json.RawMessage, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal %s payload: %w", method, err)
	}

	url := fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create %s request: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send %s request: %w", method, err)
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", method, err)
	}

	if !apiResp.OK {
		apiErr := &APIError{Code: resp.StatusCode, Description: apiResp.Description}
		if apiResp.Parameters != nil {
			apiErr.RetryAfter = apiResp.Parameters.RetryAfter
		}

		return nil, apiErr
	}

	return apiResp.Result, nil
}
