package cubepath

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const (
	// DefaultBaseURL is the default CubePath API base URL.
	DefaultBaseURL = "https://api.cubepath.com"

	// Version is the SDK version.
	Version = "0.1.0"

	defaultUserAgent = "cubepath-sdk-go/" + Version
)

// Client manages communication with the CubePath API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiToken   string
	userAgent  string

	// Retry configuration
	maxRetries   int
	retryWaitMin time.Duration
	retryWaitMax time.Duration

	// Rate limiting
	rateLimiter *rate.Limiter

	// Services
	Projects     ProjectService
	SSHKeys      SSHKeyService
	VPS          VPSService
	Baremetal    BaremetalService
	Networks     NetworkService
	FloatingIPs  FloatingIPService
	Firewall     FirewallService
	DNS          DNSService
	LoadBalancer LoadBalancerService
	CDN          CDNService
	Kubernetes   KubernetesService
	Pricing      PricingService
	DDoS         DDoSService
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithBaseURL sets a custom API base URL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithUserAgent sets a custom User-Agent header.
func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// WithMaxRetries sets the maximum number of retries for failed requests.
func WithMaxRetries(max int) ClientOption {
	return func(c *Client) {
		c.maxRetries = max
	}
}

// WithRetryWaitMin sets the minimum wait time between retries.
func WithRetryWaitMin(d time.Duration) ClientOption {
	return func(c *Client) {
		c.retryWaitMin = d
	}
}

// WithRetryWaitMax sets the maximum wait time between retries.
func WithRetryWaitMax(d time.Duration) ClientOption {
	return func(c *Client) {
		c.retryWaitMax = d
	}
}

// WithRateLimiter sets a custom rate limiter.
func WithRateLimiter(rl *rate.Limiter) ClientOption {
	return func(c *Client) {
		c.rateLimiter = rl
	}
}

// NewClient creates a new CubePath API client.
func NewClient(apiToken string, opts ...ClientOption) (*Client, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("api token is required")
	}

	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		baseURL:      DefaultBaseURL,
		apiToken:     apiToken,
		userAgent:    defaultUserAgent,
		maxRetries:   3,
		retryWaitMin: 1 * time.Second,
		retryWaitMax: 30 * time.Second,
		rateLimiter:  rate.NewLimiter(rate.Every(100*time.Millisecond), 10), // 10 req/sec
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize services
	c.Projects = &projectService{client: c}
	c.SSHKeys = &sshKeyService{client: c}
	c.VPS = &vpsService{client: c}
	c.Baremetal = &baremetalService{client: c}
	c.Networks = &networkService{client: c}
	c.FloatingIPs = &floatingIPService{client: c}
	c.Firewall = &firewallService{client: c}
	c.DNS = &dnsService{client: c}
	c.LoadBalancer = &loadBalancerService{client: c}
	c.CDN = &cdnService{client: c}
	c.Kubernetes = &kubernetesService{client: c}
	c.Pricing = &pricingService{client: c}
	c.DDoS = &ddosService{client: c}

	return c, nil
}

// newRequest creates a new HTTP request.
func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	return req, nil
}

// doRequest performs an HTTP request with retry logic.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := c.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
			}
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		req, err := c.newRequest(ctx, method, path, body)
		if err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if c.shouldRetry(resp) {
			resp.Body.Close()
			lastErr = fmt.Errorf("received status %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// shouldRetry determines if a request should be retried.
func (c *Client) shouldRetry(resp *http.Response) bool {
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	if resp.StatusCode >= 500 {
		return true
	}
	return false
}

// calculateBackoff calculates wait time with exponential backoff and jitter.
func (c *Client) calculateBackoff(attempt int) time.Duration {
	backoff := c.retryWaitMin * time.Duration(math.Pow(2, float64(attempt-1)))
	if backoff > c.retryWaitMax {
		backoff = c.retryWaitMax
	}
	jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
	return backoff + jitter
}

// handleResponse processes the HTTP response.
func (c *Client) handleResponse(resp *http.Response, result interface{}) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp)
	}

	if result == nil {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) == 0 {
		return nil
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// get performs a GET request.
func (c *Client) get(ctx context.Context, path string, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, result)
}

// post performs a POST request.
func (c *Client) post(ctx context.Context, path string, body, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, result)
}

// put performs a PUT request.
func (c *Client) put(ctx context.Context, path string, body, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, result)
}

// patch performs a PATCH request.
func (c *Client) patch(ctx context.Context, path string, body, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPatch, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, result)
}

// del performs a DELETE request.
func (c *Client) del(ctx context.Context, path string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp)
	}
	return nil
}

// delWithBody performs a DELETE request that also reads a response body.
func (c *Client) delWithBody(ctx context.Context, path string, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, result)
}

// getRaw performs a GET request and returns the raw response body.
func (c *Client) getRaw(ctx context.Context, path string) ([]byte, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseAPIError(resp)
	}

	return io.ReadAll(resp.Body)
}
