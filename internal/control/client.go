package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var ErrUnavailable = errors.New("control api unavailable")

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) Status(ctx context.Context) (StatusResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/status")
	if err != nil {
		return StatusResponse{}, err
	}

	var status StatusResponse
	if err := c.do(req, &status); err != nil {
		return StatusResponse{}, err
	}

	return status, nil
}

func (c *Client) Meta(ctx context.Context) (MetaResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/meta")
	if err != nil {
		return MetaResponse{}, err
	}

	var meta MetaResponse
	if err := c.do(req, &meta); err != nil {
		return MetaResponse{}, err
	}

	return meta, nil
}

func (c *Client) RecentLogs(ctx context.Context, limit int) (RecentLogsResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/logs/recent")
	if err != nil {
		return RecentLogsResponse{}, err
	}

	query := req.URL.Query()
	query.Set("limit", strconv.Itoa(limit))
	req.URL.RawQuery = query.Encode()

	var recent RecentLogsResponse
	if err := c.do(req, &recent); err != nil {
		return RecentLogsResponse{}, err
	}

	return recent, nil
}

func (c *Client) Stop(ctx context.Context) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/v1/stop")
	if err != nil {
		return err
	}

	var response StopResponse
	return c.do(req, &response)
}

func (c *Client) newRequest(ctx context.Context, method, path string) (*http.Request, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	targetURL := baseURL.ResolveReference(&url.URL{Path: path})
	req, err := http.NewRequestWithContext(ctx, method, targetURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return wrapUnavailableError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("control api returned status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func IsUnavailableError(err error) bool {
	return errors.Is(err, ErrUnavailable)
}

func wrapUnavailableError(err error) error {
	if isUnavailableError(err) {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	return err
}

func isUnavailableError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return isUnavailableError(urlErr.Err)
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return isUnavailableError(netErr.Err)
	}

	var syscallErr *os.SyscallError
	if errors.As(err, &syscallErr) {
		return isUnavailableError(syscallErr.Err)
	}

	return false
}
