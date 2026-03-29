package cubepath

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultAIGatewayBaseURL is the default CubePath AI Gateway base URL.
	DefaultAIGatewayBaseURL = "https://ai-gateway.cubepath.com"
)

// AIGatewayService handles communication with the CubePath AI Gateway.
// The AI Gateway provides an OpenAI-compatible API for multiple AI providers.
type AIGatewayService interface {
	// ListModels returns all available AI models with pricing and capabilities.
	ListModels(ctx context.Context) (*ModelListResponse, error)

	// ChatCompletion sends a chat completion request and returns the full response.
	ChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)

	// ChatCompletionStream sends a chat completion request and returns a stream
	// that yields chunks as they arrive via Server-Sent Events.
	// The caller must call Close() on the returned stream when done.
	ChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionStream, error)
}

// --- Request types ---

// ChatMessage represents a message in a chat conversation.
type ChatMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"`
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ToolCall represents a tool/function call in a chat message.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function details in a tool call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatCompletionRequest represents a request to the chat completions endpoint.
type ChatCompletionRequest struct {
	// Model in "provider/model_id" format, e.g. "openai/gpt-4o", "anthropic/claude-sonnet-4-20250514".
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`

	Temperature      *float64    `json:"temperature,omitempty"`
	TopP             *float64    `json:"top_p,omitempty"`
	N                *int        `json:"n,omitempty"`
	Stream           bool        `json:"stream,omitempty"`
	Stop             interface{} `json:"stop,omitempty"`
	MaxTokens        *int        `json:"max_tokens,omitempty"`
	PresencePenalty  *float64    `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64    `json:"frequency_penalty,omitempty"`
	User             string      `json:"user,omitempty"`
	Tools            []Tool      `json:"tools,omitempty"`
	ToolChoice       interface{} `json:"tool_choice,omitempty"`
	ResponseFormat   interface{} `json:"response_format,omitempty"`
}

// Tool represents a tool definition for function calling.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a function available as a tool.
type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// --- Response types ---

// ChatCompletionResponse represents a non-streaming chat completion response.
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   *CompletionUsage       `json:"usage,omitempty"`
}

// ChatCompletionChoice represents a single choice in a completion response.
type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason *string     `json:"finish_reason"`
}

// CompletionUsage represents token usage information.
type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// --- Streaming types ---

// ChatCompletionChunk represents a single chunk in a streaming response.
type ChatCompletionChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []ChatCompletionDelta `json:"choices"`
	Usage   *CompletionUsage    `json:"usage,omitempty"`
}

// ChatCompletionDelta represents a delta choice in a streaming chunk.
type ChatCompletionDelta struct {
	Index        int          `json:"index"`
	Delta        DeltaContent `json:"delta"`
	FinishReason *string      `json:"finish_reason"`
}

// DeltaContent represents the incremental content in a streaming chunk.
type DeltaContent struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChatCompletionStream reads streaming chat completion chunks from the API.
// It must be closed after use.
type ChatCompletionStream struct {
	reader  *bufio.Reader
	body    io.ReadCloser
	done    bool
	mu      sync.Mutex
}

// Recv reads the next chunk from the stream.
// Returns io.EOF when the stream is complete (after receiving [DONE]).
func (s *ChatCompletionStream) Recv() (*ChatCompletionChunk, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return nil, io.EOF
	}

	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			s.done = true
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, fmt.Errorf("error reading stream: %w", err)
		}

		line = strings.TrimSpace(line)

		// Skip empty lines (SSE separator)
		if line == "" {
			continue
		}

		// Skip SSE comments
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE data lines
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			s.done = true
			return nil, io.EOF
		}

		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("error unmarshaling chunk: %w", err)
		}

		return &chunk, nil
	}
}

// Close closes the stream and releases the underlying connection.
func (s *ChatCompletionStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.done = true
	return s.body.Close()
}

// --- Model types ---

// ModelListResponse represents the response from the list models endpoint.
type ModelListResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// ModelInfo represents information about a single AI model.
type ModelInfo struct {
	ID           string            `json:"id"`
	Object       string            `json:"object"`
	OwnedBy      string            `json:"owned_by"`
	Pricing      ModelPricing      `json:"pricing"`
	Capabilities ModelCapabilities `json:"capabilities"`
	Limits       ModelLimits       `json:"limits"`
}

// ModelPricing represents the pricing information for a model.
type ModelPricing struct {
	InputPerMillionTokens  string `json:"input_per_million_tokens"`
	OutputPerMillionTokens string `json:"output_per_million_tokens"`
	Currency               string `json:"currency"`
}

// ModelCapabilities describes what features a model supports.
type ModelCapabilities struct {
	Streaming bool `json:"streaming"`
	Vision    bool `json:"vision"`
	Tools     bool `json:"tools"`
}

// ModelLimits describes the token limits for a model.
type ModelLimits struct {
	MaxContextTokens int `json:"max_context_tokens"`
	MaxOutputTokens  int `json:"max_output_tokens"`
}

// --- Service implementation ---

type aiGatewayService struct {
	client  *Client
	baseURL string
}

func (s *aiGatewayService) ListModels(ctx context.Context) (*ModelListResponse, error) {
	var result ModelListResponse
	if err := s.doGet(ctx, "/models", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *aiGatewayService) ChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// Ensure stream is false for non-streaming requests
	req.Stream = false

	var result ChatCompletionResponse
	if err := s.doPost(ctx, "/chat/completions", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *aiGatewayService) ChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionStream, error) {
	req.Stream = true

	resp, err := s.doRequestRaw(ctx, http.MethodPost, "/chat/completions", req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, parseAPIError(resp)
	}

	return &ChatCompletionStream{
		reader: bufio.NewReader(resp.Body),
		body:   resp.Body,
	}, nil
}

// doRequestRaw performs an HTTP request against the AI Gateway and returns the raw response.
// The caller is responsible for closing the response body.
func (s *aiGatewayService) doRequestRaw(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	c := s.client

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

		req, err := c.newRequestWithURL(ctx, method, s.baseURL+path, body)
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

// doGet performs a GET request against the AI Gateway.
func (s *aiGatewayService) doGet(ctx context.Context, path string, result interface{}) error {
	resp, err := s.doRequestRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return s.client.handleResponse(resp, result)
}

// doPost performs a POST request against the AI Gateway.
func (s *aiGatewayService) doPost(ctx context.Context, path string, body, result interface{}) error {
	resp, err := s.doRequestRaw(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return s.client.handleResponse(resp, result)
}
