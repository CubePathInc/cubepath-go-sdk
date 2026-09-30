package cubepath

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// graphQLRequest is the body of a POST /graphql call.
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// graphQLError is one entry of the "errors" list of a GraphQL response.
type graphQLError struct {
	Message    string `json:"message"`
	Extensions struct {
		Code string `json:"code"`
	} `json:"extensions"`
}

// graphQL runs a query against POST /graphql and decodes its "data" object into result.
//
// Metrics (VPS, baremetal, NAT gateways...) are only served through GraphQL. A GraphQL error
// is returned as an *APIError; NOT_FOUND maps to HTTP 404 so IsNotFound keeps working.
func (c *Client) graphQL(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []graphQLError  `json:"errors"`
	}
	if err := c.post(ctx, "/graphql", &graphQLRequest{Query: query, Variables: variables}, &envelope); err != nil {
		return err
	}
	if len(envelope.Errors) > 0 {
		status := http.StatusBadRequest
		messages := make([]string, 0, len(envelope.Errors))
		for _, e := range envelope.Errors {
			messages = append(messages, e.Message)
			switch e.Extensions.Code {
			case "NOT_FOUND":
				status = http.StatusNotFound
			case "FORBIDDEN":
				status = http.StatusForbidden
			case "UNAUTHENTICATED":
				status = http.StatusUnauthorized
			}
		}
		return &APIError{StatusCode: status, Message: "GraphQL error", Detail: strings.Join(messages, "; ")}
	}
	if result == nil || len(envelope.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, result); err != nil {
		return fmt.Errorf("failed to unmarshal GraphQL data: %w", err)
	}
	return nil
}
