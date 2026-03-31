package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bob/internal/protocol"
)

type Client struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

func New(endpoint, token string, timeout time.Duration) *Client {
	return &Client{
		endpoint: strings.TrimRight(endpoint, "/"),
		token:    token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Open(ctx context.Context, rawURL string) (*protocol.OpenResponse, error) {
	host, _ := os.Hostname()
	cwd, _ := os.Getwd()
	nonce, err := randomNonce()
	if err != nil {
		return nil, err
	}

	reqBody := protocol.OpenRequest{
		Version: protocol.CurrentVersion,
		Action:  protocol.ActionOpenURL,
		URL:     rawURL,
		Source: protocol.Source{
			App:  filepath.Base(os.Args[0]),
			Host: host,
			CWD:  cwd,
		},
		Timestamp: time.Now().Unix(),
		Nonce:     nonce,
	}

	requestURL, err := joinEndpoint(c.endpoint, "/open")
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out protocol.OpenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *Client) Health(ctx context.Context) (*protocol.HealthResponse, error) {
	requestURL, err := joinEndpoint(c.endpoint, "/healthz")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected health status: %s", resp.Status)
	}

	var out protocol.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

func joinEndpoint(base, path string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("endpoint must be a full URL: %s", base)
	}

	ref, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return parsed.ResolveReference(ref).String(), nil
}

func randomNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
