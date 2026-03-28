package cubepath

import (
	"context"
	"fmt"
)

// SSHKeyService handles communication with the SSH key related methods of the CubePath API.
type SSHKeyService interface {
	Create(ctx context.Context, req *CreateSSHKeyRequest) (*SSHKey, error)
	List(ctx context.Context) ([]SSHKey, error)
	Delete(ctx context.Context, keyID int) error
}

// SSHKey represents an SSH key.
type SSHKey struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	SSHKey      string `json:"ssh_key"`
	Fingerprint string `json:"fingerprint"`
	KeyType     string `json:"key_type"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// CreateSSHKeyRequest represents a request to create an SSH key.
type CreateSSHKeyRequest struct {
	Name   string `json:"name"`
	SSHKey string `json:"ssh_key"`
}

type sshKeyService struct {
	client *Client
}

func (s *sshKeyService) Create(ctx context.Context, req *CreateSSHKeyRequest) (*SSHKey, error) {
	var key SSHKey
	if err := s.client.post(ctx, "/sshkey/create", req, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

func (s *sshKeyService) List(ctx context.Context) ([]SSHKey, error) {
	var result struct {
		SSHKeys []SSHKey `json:"sshkeys"`
	}
	if err := s.client.get(ctx, "/sshkey/user/sshkeys", &result); err != nil {
		return nil, err
	}
	return result.SSHKeys, nil
}

func (s *sshKeyService) Delete(ctx context.Context, keyID int) error {
	return s.client.del(ctx, fmt.Sprintf("/sshkey/%d", keyID))
}
