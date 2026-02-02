package device

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

// FirmwareStatus represents the current status of a firmware update
type FirmwareStatus string

const (
	FirmwareStatusIdle                   FirmwareStatus = "Idle"
	FirmwareStatusDownloading            FirmwareStatus = "Downloading"
	FirmwareStatusDownloaded             FirmwareStatus = "Downloaded"
	FirmwareStatusDownloadFailed         FirmwareStatus = "DownloadFailed"
	FirmwareStatusDownloadScheduled      FirmwareStatus = "DownloadScheduled"
	FirmwareStatusDownloadPaused         FirmwareStatus = "DownloadPaused"
	FirmwareStatusInstalling             FirmwareStatus = "Installing"
	FirmwareStatusInstalled              FirmwareStatus = "Installed"
	FirmwareStatusInstallationFailed     FirmwareStatus = "InstallationFailed"
	FirmwareStatusInstallRebooting       FirmwareStatus = "InstallRebooting"
	FirmwareStatusInstallScheduled       FirmwareStatus = "InstallScheduled"
	FirmwareStatusInstallVerificationFailed FirmwareStatus = "InstallVerificationFailed"
	FirmwareStatusInvalidSignature       FirmwareStatus = "InvalidSignature"
	FirmwareStatusSignatureVerified      FirmwareStatus = "SignatureVerified"
)

// FirmwareUpdate represents a firmware update request
type FirmwareUpdate struct {
	ID              string         `json:"id"`
	ChargePointID   string         `json:"charge_point_id"`
	RequestID       int            `json:"request_id"`
	FirmwareURL     string         `json:"firmware_url"`
	Version         string         `json:"version"`
	RetrieveDateTime time.Time     `json:"retrieve_datetime"`
	InstallDateTime *time.Time     `json:"install_datetime,omitempty"`
	Status          FirmwareStatus `json:"status"`
	Progress        int            `json:"progress"` // 0-100
	ErrorMessage    string         `json:"error_message,omitempty"`
	Retries         int            `json:"retries"`
	MaxRetries      int            `json:"max_retries"`
	RetryInterval   int            `json:"retry_interval"` // seconds
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
}

// FirmwareService handles firmware update operations
type FirmwareService struct {
	ocppServer  ports.OCPPCommandService
	mq          ports.MessageQueue
	log         *zap.Logger

	// Track ongoing updates
	updates     map[string]*FirmwareUpdate // key: chargePointID
	updatesByID map[string]*FirmwareUpdate // key: update ID
	mu          sync.RWMutex
}

// NewFirmwareService creates a new firmware service
func NewFirmwareService(ocppServer ports.OCPPCommandService, mq ports.MessageQueue, log *zap.Logger) *FirmwareService {
	return &FirmwareService{
		ocppServer:  ocppServer,
		mq:          mq,
		log:         log,
		updates:     make(map[string]*FirmwareUpdate),
		updatesByID: make(map[string]*FirmwareUpdate),
	}
}

// UpdateFirmwareRequest represents a request to update firmware
type UpdateFirmwareRequest struct {
	ChargePointID     string     `json:"charge_point_id"`
	FirmwareURL       string     `json:"firmware_url"`
	Version           string     `json:"version"`
	RetrieveDateTime  *time.Time `json:"retrieve_datetime,omitempty"`
	InstallDateTime   *time.Time `json:"install_datetime,omitempty"`
	Retries           *int       `json:"retries,omitempty"`
	RetryInterval     *int       `json:"retry_interval,omitempty"` // seconds
	SigningCertificate string    `json:"signing_certificate,omitempty"`
	Signature         string     `json:"signature,omitempty"`
}

// UpdateFirmware initiates a firmware update on a charge point
func (s *FirmwareService) UpdateFirmware(ctx context.Context, req *UpdateFirmwareRequest) (*FirmwareUpdate, error) {
	// Check if there's already an update in progress
	s.mu.RLock()
	existingUpdate, exists := s.updates[req.ChargePointID]
	s.mu.RUnlock()

	if exists && !isTerminalStatus(existingUpdate.Status) {
		return nil, fmt.Errorf("firmware update already in progress for %s", req.ChargePointID)
	}

	// Set defaults
	retrieveTime := time.Now()
	if req.RetrieveDateTime != nil {
		retrieveTime = *req.RetrieveDateTime
	}

	retries := 3
	if req.Retries != nil {
		retries = *req.Retries
	}

	retryInterval := 60 // 60 seconds
	if req.RetryInterval != nil {
		retryInterval = *req.RetryInterval
	}

	// Create update record
	update := &FirmwareUpdate{
		ID:               uuid.New().String(),
		ChargePointID:    req.ChargePointID,
		RequestID:        int(time.Now().UnixNano() % 1000000),
		FirmwareURL:      req.FirmwareURL,
		Version:          req.Version,
		RetrieveDateTime: retrieveTime,
		InstallDateTime:  req.InstallDateTime,
		Status:           FirmwareStatusIdle,
		MaxRetries:       retries,
		RetryInterval:    retryInterval,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Send OCPP command
	var err error
	if req.SigningCertificate != "" && req.Signature != "" {
		err = s.ocppServer.UpdateFirmwareSigned(
			ctx,
			req.ChargePointID,
			req.FirmwareURL,
			retrieveTime.Format(time.RFC3339),
			req.SigningCertificate,
			req.Signature,
			&retries,
			&retryInterval,
		)
	} else {
		err = s.ocppServer.UpdateFirmware(
			ctx,
			req.ChargePointID,
			req.FirmwareURL,
			retrieveTime.Format(time.RFC3339),
			req.InstallDateTime,
			&retries,
			&retryInterval,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to send firmware update command: %w", err)
	}

	update.Status = FirmwareStatusDownloadScheduled

	// Store update
	s.mu.Lock()
	s.updates[req.ChargePointID] = update
	s.updatesByID[update.ID] = update
	s.mu.Unlock()

	// Publish event
	if s.mq != nil {
		s.mq.Publish("firmware.update.started", map[string]interface{}{
			"update_id":       update.ID,
			"charge_point_id": req.ChargePointID,
			"firmware_url":    req.FirmwareURL,
			"version":         req.Version,
		})
	}

	s.log.Info("Firmware update initiated",
		zap.String("updateID", update.ID),
		zap.String("chargePointID", req.ChargePointID),
		zap.String("version", req.Version),
	)

	return update, nil
}

// GetFirmwareStatus returns the current firmware update status for a charge point
func (s *FirmwareService) GetFirmwareStatus(ctx context.Context, chargePointID string) (*FirmwareUpdate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	update, exists := s.updates[chargePointID]
	if !exists {
		return nil, nil
	}

	return update, nil
}

// GetFirmwareUpdate returns a firmware update by ID
func (s *FirmwareService) GetFirmwareUpdate(ctx context.Context, updateID string) (*FirmwareUpdate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	update, exists := s.updatesByID[updateID]
	if !exists {
		return nil, fmt.Errorf("firmware update %s not found", updateID)
	}

	return update, nil
}

// CancelFirmwareUpdate attempts to cancel a firmware update
func (s *FirmwareService) CancelFirmwareUpdate(ctx context.Context, chargePointID string) error {
	s.mu.Lock()
	update, exists := s.updates[chargePointID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("no firmware update found for %s", chargePointID)
	}

	// Can only cancel if not yet installing
	if update.Status == FirmwareStatusInstalling ||
		update.Status == FirmwareStatusInstallRebooting {
		s.mu.Unlock()
		return fmt.Errorf("cannot cancel firmware update in status %s", update.Status)
	}

	update.Status = FirmwareStatusIdle
	update.UpdatedAt = time.Now()
	s.mu.Unlock()

	// Note: OCPP 2.0.1 doesn't have a direct cancel firmware command
	// The charge point will stop on its own if we don't send new requests

	// Publish event
	if s.mq != nil {
		s.mq.Publish("firmware.update.cancelled", map[string]interface{}{
			"update_id":       update.ID,
			"charge_point_id": chargePointID,
		})
	}

	s.log.Info("Firmware update cancelled",
		zap.String("updateID", update.ID),
		zap.String("chargePointID", chargePointID),
	)

	return nil
}

// HandleFirmwareStatusNotification processes firmware status updates from charge points
func (s *FirmwareService) HandleFirmwareStatusNotification(chargePointID string, status string, requestID *int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	update, exists := s.updates[chargePointID]
	if !exists {
		// Create a placeholder update for unsolicited status
		update = &FirmwareUpdate{
			ID:            uuid.New().String(),
			ChargePointID: chargePointID,
			CreatedAt:     time.Now(),
		}
		s.updates[chargePointID] = update
		s.updatesByID[update.ID] = update
	}

	prevStatus := update.Status
	update.Status = FirmwareStatus(status)
	update.UpdatedAt = time.Now()

	if requestID != nil {
		update.RequestID = *requestID
	}

	// Update progress based on status
	switch update.Status {
	case FirmwareStatusDownloading:
		update.Progress = 25
	case FirmwareStatusDownloaded:
		update.Progress = 50
	case FirmwareStatusInstalling:
		update.Progress = 75
	case FirmwareStatusInstalled:
		update.Progress = 100
		now := time.Now()
		update.CompletedAt = &now
	case FirmwareStatusDownloadFailed, FirmwareStatusInstallationFailed,
		FirmwareStatusInvalidSignature, FirmwareStatusInstallVerificationFailed:
		update.ErrorMessage = string(update.Status)
	}

	// Publish progress event
	if s.mq != nil {
		eventType := "firmware.update.progress"
		if isTerminalStatus(update.Status) {
			if update.Status == FirmwareStatusInstalled {
				eventType = "firmware.update.completed"
			} else {
				eventType = "firmware.update.failed"
			}
		}

		s.mq.Publish(eventType, map[string]interface{}{
			"update_id":       update.ID,
			"charge_point_id": chargePointID,
			"status":          status,
			"progress":        update.Progress,
			"error_message":   update.ErrorMessage,
		})
	}

	s.log.Info("Firmware status notification",
		zap.String("chargePointID", chargePointID),
		zap.String("prevStatus", string(prevStatus)),
		zap.String("newStatus", status),
		zap.Int("progress", update.Progress),
	)

	return nil
}

// GetActiveUpdates returns all active firmware updates
func (s *FirmwareService) GetActiveUpdates() []*FirmwareUpdate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*FirmwareUpdate
	for _, update := range s.updates {
		if !isTerminalStatus(update.Status) {
			active = append(active, update)
		}
	}

	return active
}

// GetRecentUpdates returns recent firmware updates (last 24 hours)
func (s *FirmwareService) GetRecentUpdates() []*FirmwareUpdate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	var recent []*FirmwareUpdate

	for _, update := range s.updatesByID {
		if update.CreatedAt.After(cutoff) {
			recent = append(recent, update)
		}
	}

	return recent
}

// CleanupOldUpdates removes updates older than retention period
func (s *FirmwareService) CleanupOldUpdates(retention time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-retention)
	removed := 0

	for id, update := range s.updatesByID {
		if update.CreatedAt.Before(cutoff) && isTerminalStatus(update.Status) {
			delete(s.updatesByID, id)
			delete(s.updates, update.ChargePointID)
			removed++
		}
	}

	return removed
}

// Helper function to check if status is terminal (completed or failed)
func isTerminalStatus(status FirmwareStatus) bool {
	switch status {
	case FirmwareStatusInstalled,
		FirmwareStatusDownloadFailed,
		FirmwareStatusInstallationFailed,
		FirmwareStatusInvalidSignature,
		FirmwareStatusInstallVerificationFailed,
		FirmwareStatusIdle:
		return true
	default:
		return false
	}
}
