package cubepath

import (
	"context"
	"fmt"
)

// VPSBackupService handles communication with the VPS backup related methods of the CubePath API.
type VPSBackupService interface {
	List(ctx context.Context, vpsID int) ([]VPSBackup, error)
	Create(ctx context.Context, vpsID int, req *CreateVPSBackupRequest) error
	Restore(ctx context.Context, vpsID, backupID int) error
	Delete(ctx context.Context, vpsID, backupID int) error
	GetSettings(ctx context.Context, vpsID int) (*VPSBackupSettings, error)
	UpdateSettings(ctx context.Context, vpsID int, req *UpdateVPSBackupSettingsRequest) error
}

// VPSBackup represents a VPS backup.
type VPSBackup struct {
	ID         int     `json:"id"`
	BackupType string  `json:"backup_type"`
	Status     string  `json:"status"`
	Progress   int     `json:"progress"`
	SizeGB     float64 `json:"size_gb"`
	Notes      string  `json:"notes"`
	CreatedAt  string  `json:"created_at"`
}

// VPSBackupSettings represents backup settings for a VPS.
type VPSBackupSettings struct {
	Enabled       bool `json:"enabled"`
	ScheduleHour  int  `json:"schedule_hour"`
	RetentionDays int  `json:"retention_days"`
	MaxBackups    int  `json:"max_backups"`
}

// CreateVPSBackupRequest represents a request to create a VPS backup.
type CreateVPSBackupRequest struct {
	Notes string `json:"notes,omitempty"`
}

// UpdateVPSBackupSettingsRequest represents a request to update VPS backup settings.
type UpdateVPSBackupSettingsRequest struct {
	Enabled       bool `json:"enabled"`
	ScheduleHour  int  `json:"schedule_hour"`
	RetentionDays int  `json:"retention_days"`
	MaxBackups    int  `json:"max_backups"`
}

type vpsBackupService struct {
	client *Client
}

func (s *vpsBackupService) List(ctx context.Context, vpsID int) ([]VPSBackup, error) {
	var result struct {
		Backups []VPSBackup `json:"backups"`
	}
	if err := s.client.get(ctx, fmt.Sprintf("/vps/%d/backups", vpsID), &result); err != nil {
		return nil, err
	}
	return result.Backups, nil
}

func (s *vpsBackupService) Create(ctx context.Context, vpsID int, req *CreateVPSBackupRequest) error {
	body := map[string]interface{}{}
	if req != nil && req.Notes != "" {
		body["notes"] = req.Notes
	}
	return s.client.post(ctx, fmt.Sprintf("/vps/%d/backups", vpsID), body, nil)
}

func (s *vpsBackupService) Restore(ctx context.Context, vpsID, backupID int) error {
	body := map[string]interface{}{
		"confirm": true,
	}
	return s.client.post(ctx, fmt.Sprintf("/vps/%d/backups/%d/restore", vpsID, backupID), body, nil)
}

func (s *vpsBackupService) Delete(ctx context.Context, vpsID, backupID int) error {
	return s.client.del(ctx, fmt.Sprintf("/vps/%d/backups/%d", vpsID, backupID))
}

func (s *vpsBackupService) GetSettings(ctx context.Context, vpsID int) (*VPSBackupSettings, error) {
	var settings VPSBackupSettings
	if err := s.client.get(ctx, fmt.Sprintf("/vps/%d/backup/settings", vpsID), &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

func (s *vpsBackupService) UpdateSettings(ctx context.Context, vpsID int, req *UpdateVPSBackupSettingsRequest) error {
	return s.client.put(ctx, fmt.Sprintf("/vps/%d/backup/settings", vpsID), req, nil)
}
