package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/YehhiAleksandra/oracle-gateway/internal/config"
)

type Client struct {
	http    *http.Client
	cfg     config.Config
	baseURL string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompleteRequest struct {
	System    string `json:"system"`
	User      string `json:"user"`
	MaxTokens int    `json:"max_tokens"`
	Task      string `json:"task"`
}

type VisionRequest struct {
	System    string `json:"system"`
	User      string `json:"user"`
	ImageURL  string `json:"image_url"`
	MaxTokens int    `json:"max_tokens"`
	Task      string `json:"task"`
}

type CompleteResponse struct {
	Text  string `json:"text"`
	Model string `json:"model"`
	MS    int64  `json:"elapsed_ms"`
}

type imagePart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
}

type userMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type chatPayload struct {
	Model       string    `json:"model"`
	Messages    []any     `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			Reasoning string `json:"reasoning"`
		} `json:"message"`
	} `json:"choices"`
}

func New(cfg config.Config) *Client {
	return &Client{
		cfg:     cfg,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		http: &http.Client{
			Timeout: time.Duration(cfg.ReadingTimeoutSec+5) * time.Second,
		},
	}
}

func (c *Client) Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error) {
	if c.cfg.APIKey == "" {
		return CompleteResponse{}, errors.New("OPENAI_API_KEY is not set")
	}
	if strings.TrimSpace(req.System) == "" || strings.TrimSpace(req.User) == "" {
		return CompleteResponse{}, errors.New("system and user are required")
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 900
	}

	var lastErr error
	started := time.Now()
	for _, model := range c.cfg.ModelsForTask(req.Task) {
		text, err := c.callModel(ctx, model, req.System, req.User, maxTokens)
		if err == nil {
			return CompleteResponse{
				Text:  text,
				Model: model,
				MS:    time.Since(started).Milliseconds(),
			}, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no models configured")
	}
	return CompleteResponse{}, lastErr
}

func (c *Client) Ping(ctx context.Context) (CompleteResponse, error) {
	return c.Complete(ctx, CompleteRequest{
		System:    "Reply with exactly: OK",
		User:      "ping",
		MaxTokens: 16,
		Task:      "ping",
	})
}

func (c *Client) CompleteVision(ctx context.Context, req VisionRequest) (CompleteResponse, error) {
	if c.cfg.APIKey == "" {
		return CompleteResponse{}, errors.New("OPENAI_API_KEY is not set")
	}
	if strings.TrimSpace(req.System) == "" || strings.TrimSpace(req.User) == "" {
		return CompleteResponse{}, errors.New("system and user are required")
	}
	if strings.TrimSpace(req.ImageURL) == "" {
		return CompleteResponse{}, errors.New("image_url is required")
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1200
	}

	var lastErr error
	started := time.Now()
	for _, model := range c.cfg.VisionModelsForTask(req.Task) {
		text, err := c.callVisionModel(ctx, model, req.System, req.User, req.ImageURL, maxTokens)
		if err == nil {
			return CompleteResponse{
				Text:  text,
				Model: model,
				MS:    time.Since(started).Milliseconds(),
			}, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no vision models configured")
	}
	return CompleteResponse{}, lastErr
}

func (c *Client) callVisionModel(ctx context.Context, model, system, user, imageURL string, maxTokens int) (string, error) {
	timeout := time.Duration(c.cfg.ReadingTimeoutSec) * time.Second
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		text, err := c.doVisionRequest(callCtx, model, system, user, imageURL, maxTokens)
		cancel()
		if err == nil {
			return text, nil
		}
		lastErr = err
		if !isRetryable(err) || attempt == 1 {
			break
		}
		time.Sleep(retryDelay(err))
	}
	return "", fmt.Errorf("%s: %w", model, lastErr)
}

func (c *Client) callModel(ctx context.Context, model, system, user string, maxTokens int) (string, error) {
	timeout := time.Duration(c.cfg.ReadingTimeoutSec) * time.Second
	if maxTokens <= 32 {
		timeout = time.Duration(c.cfg.PerModelTimeoutSec) * time.Second
	}

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		text, err := c.doRequest(callCtx, model, system, user, maxTokens)
		cancel()
		if err == nil {
			return text, nil
		}
		lastErr = err
		if !isRetryable(err) || attempt == 1 {
			break
		}
		time.Sleep(retryDelay(err))
	}
	return "", fmt.Errorf("%s: %w", model, lastErr)
}

func (c *Client) doRequest(ctx context.Context, model, system, user string, maxTokens int) (string, error) {
	body, err := json.Marshal(chatPayload{
		Model: model,
		Messages: []any{
			Message{Role: "system", Content: system},
			Message{Role: "user", Content: user},
		},
		Temperature: 0.85,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/YehhiAleksandra/oracle-gateway")
	req.Header.Set("X-Title", "Oracle Gateway")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", &rateLimitError{body: string(raw)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("upstream HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("empty choices")
	}
	msg := parsed.Choices[0].Message
	text := strings.TrimSpace(msg.Content)
	if text == "" {
		text = strings.TrimSpace(msg.Reasoning)
	}
	text = sanitize(text)
	if text == "" {
		return "", errors.New("empty response text")
	}
	return text, nil
}

func (c *Client) doVisionRequest(ctx context.Context, model, system, user, imageURL string, maxTokens int) (string, error) {
	body, err := json.Marshal(chatPayload{
		Model: model,
		Messages: []any{
			Message{Role: "system", Content: system},
			userMessage{
				Role: "user",
				Content: []imagePart{
					{Type: "text", Text: user},
					{Type: "image_url", ImageURL: &struct {
						URL string `json:"url"`
					}{URL: imageURL}},
				},
			},
		},
		Temperature: 0.85,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/YehhiAleksandra/oracle-gateway")
	req.Header.Set("X-Title", "Oracle Gateway")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", &rateLimitError{body: string(raw)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("upstream HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("empty choices")
	}
	msg := parsed.Choices[0].Message
	text := strings.TrimSpace(msg.Content)
	if text == "" {
		text = strings.TrimSpace(msg.Reasoning)
	}
	text = sanitize(text)
	if text == "" {
		return "", errors.New("empty response text")
	}
	return text, nil
}

type rateLimitError struct {
	body string
}

func (e *rateLimitError) Error() string {
	return "rate limited: " + truncate(e.body, 120)
}

func isRetryable(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var rl *rateLimitError
	return errors.As(err, &rl)
}

func retryDelay(err error) time.Duration {
	return 30 * time.Second
}

func sanitize(text string) string {
	lower := strings.ToLower(text)
	prefixes := []string{"we need", "okay,", "the user", "let me"}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			for _, marker := range []string{"**", "###", "\n\n"} {
				if idx := strings.Index(text, marker); idx > 15 {
					return strings.TrimSpace(text[idx:])
				}
			}
		}
	}
	return text
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
