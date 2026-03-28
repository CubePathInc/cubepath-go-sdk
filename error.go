package cubepath

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// APIError represents an error returned by the CubePath API.
type APIError struct {
	StatusCode int
	Message    string
	Detail     string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("API Error (HTTP %d): %s - %s", e.StatusCode, e.Message, e.Detail)
	}
	return fmt.Sprintf("API Error (HTTP %d): %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error is a 404 Not Found.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsConflict returns true if the error is a 409 Conflict.
func (e *APIError) IsConflict() bool {
	return e.StatusCode == http.StatusConflict
}

// IsRateLimited returns true if the error is a 429 Too Many Requests.
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// IsBadRequest returns true if the error is a 400 Bad Request.
func (e *APIError) IsBadRequest() bool {
	return e.StatusCode == http.StatusBadRequest
}

// IsServerError returns true if the error is a 5xx server error.
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// IsNotFound checks if an error is a CubePath API 404 Not Found error.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsNotFound()
	}
	return false
}

// IsConflict checks if an error is a CubePath API 409 Conflict error.
func IsConflict(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsConflict()
	}
	return false
}

// IsRateLimited checks if an error is a CubePath API 429 Too Many Requests error.
func IsRateLimited(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsRateLimited()
	}
	return false
}

// IsBadRequest checks if an error is a CubePath API 400 Bad Request error.
func IsBadRequest(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsBadRequest()
	}
	return false
}

// parseAPIError parses an HTTP response into an APIError.
func parseAPIError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
			Detail:     fmt.Sprintf("failed to read error response: %v", err),
		}
	}

	// Try to parse FastAPI error format: {"detail": "error message"}
	var apiErr struct {
		Detail interface{} `json:"detail"`
	}

	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Detail != nil {
		switch v := apiErr.Detail.(type) {
		case string:
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    http.StatusText(resp.StatusCode),
				Detail:     v,
			}
		default:
			detailBytes, err := json.Marshal(v)
			if err != nil {
				return &APIError{
					StatusCode: resp.StatusCode,
					Message:    http.StatusText(resp.StatusCode),
					Detail:     fmt.Sprintf("%v", v),
				}
			}
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    http.StatusText(resp.StatusCode),
				Detail:     string(detailBytes),
			}
		}
	}

	// Fallback to raw body
	if len(body) > 0 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
			Detail:     string(body),
		}
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    http.StatusText(resp.StatusCode),
	}
}
