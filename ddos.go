package cubepath

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// DDoSService handles communication with the DDoS attack related methods of the CubePath API.
type DDoSService interface {
	// ListAttacks returns the recent attacks against the IPs of the organization. It returns
	// an empty list when there are none.
	ListAttacks(ctx context.Context) ([]DDoSAttack, error)
	// GetAttackDetails returns the details the detection platform recorded for an attack
	// (attack vectors, sources and so on), as raw JSON.
	GetAttackDetails(ctx context.Context, attackID int) (json.RawMessage, error)
	// GetAttackTrafficGraph returns the traffic time series of an attack, as raw JSON.
	GetAttackTrafficGraph(ctx context.Context, attackID int) (json.RawMessage, error)
}

// DDoSAttack represents a DDoS attack event. Duration and the peaks may be decimals.
type DDoSAttack struct {
	AttackID          int     `json:"attack_id"`
	IPAddress         string  `json:"ip_address"`
	StartTime         string  `json:"start_time"`
	Duration          float64 `json:"duration"`
	PacketsSecondPeak float64 `json:"packets_second_peak"`
	GbpsPeak          float64 `json:"gbps_peak"`
	// Deprecated: not returned by the API; use GbpsPeak.
	BytesSecondPeak int    `json:"bytes_second_peak"`
	Status          string `json:"status"`
	Description     string `json:"description"`
}

type ddosService struct {
	client *Client
}

func (s *ddosService) ListAttacks(ctx context.Context) ([]DDoSAttack, error) {
	data, err := s.client.getRaw(ctx, "/ddos-attacks/attacks")
	if err != nil {
		return nil, err
	}
	// Without attacks the API answers {"detail": "No recent DDoS attacks ..."} instead of a list.
	if trimmed := bytes.TrimSpace(data); len(trimmed) == 0 || trimmed[0] != '[' {
		return []DDoSAttack{}, nil
	}
	var attacks []DDoSAttack
	if err := json.Unmarshal(data, &attacks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return attacks, nil
}

func (s *ddosService) GetAttackDetails(ctx context.Context, attackID int) (json.RawMessage, error) {
	data, err := s.client.getRaw(ctx, fmt.Sprintf("/ddos-attacks/attacks/%d/details", attackID))
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func (s *ddosService) GetAttackTrafficGraph(ctx context.Context, attackID int) (json.RawMessage, error) {
	data, err := s.client.getRaw(ctx, fmt.Sprintf("/ddos-attacks/attacks/%d/traffic-graph", attackID))
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
