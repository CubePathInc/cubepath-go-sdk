package cubepath

import "context"

// DDoSService handles communication with the DDoS attack related methods of the CubePath API.
type DDoSService interface {
	ListAttacks(ctx context.Context) ([]DDoSAttack, error)
}

// DDoSAttack represents a DDoS attack event.
type DDoSAttack struct {
	AttackID          int    `json:"attack_id"`
	IPAddress         string `json:"ip_address"`
	StartTime         string `json:"start_time"`
	Duration          int    `json:"duration"`
	PacketsSecondPeak int    `json:"packets_second_peak"`
	BytesSecondPeak   int    `json:"bytes_second_peak"`
	Status            string `json:"status"`
	Description       string `json:"description"`
}

type ddosService struct {
	client *Client
}

func (s *ddosService) ListAttacks(ctx context.Context) ([]DDoSAttack, error) {
	var attacks []DDoSAttack
	if err := s.client.get(ctx, "/ddos-attacks/attacks", &attacks); err != nil {
		return nil, err
	}
	return attacks, nil
}
