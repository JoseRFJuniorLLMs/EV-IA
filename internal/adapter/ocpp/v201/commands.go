package v201

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// --- Remote Start/Stop Transaction ---

// RemoteStartTransaction requests a charge point to start a transaction
func (s *Server) RemoteStartTransaction(ctx context.Context, chargePointID string, idToken string, evseID *int, chargingProfile *ChargingProfile) (*RequestStartTransactionResponse, error) {
	req := RequestStartTransactionRequest{
		IdToken: IdToken{
			IdToken: idToken,
			Type:    "ISO14443",
		},
		RemoteStartId: int(time.Now().UnixNano() % 1000000),
		EvseId:        evseID,
		ChargingProfile: chargingProfile,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "RequestStartTransaction", req)
	if err != nil {
		return nil, fmt.Errorf("remote start transaction failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("remote start rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response RequestStartTransactionResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// RemoteStopTransaction requests a charge point to stop a transaction
func (s *Server) RemoteStopTransaction(ctx context.Context, chargePointID, transactionID string) (*RequestStopTransactionResponse, error) {
	req := RequestStopTransactionRequest{
		TransactionId: transactionID,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "RequestStopTransaction", req)
	if err != nil {
		return nil, fmt.Errorf("remote stop transaction failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("remote stop rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response RequestStopTransactionResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// --- Reset ---

// Reset requests a charge point to reset
func (s *Server) Reset(ctx context.Context, chargePointID string, resetType string, evseID *int) (*ResetResponse, error) {
	req := ResetRequest{
		Type:   resetType, // "Immediate" or "OnIdle"
		EvseId: evseID,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "Reset", req)
	if err != nil {
		return nil, fmt.Errorf("reset failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("reset rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response ResetResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// --- Trigger Message ---

// TriggerMessage requests a charge point to send a specific message
func (s *Server) TriggerMessage(ctx context.Context, chargePointID, requestedMessage string, evse *Evse) (*TriggerMessageResponse, error) {
	req := TriggerMessageRequest{
		RequestedMessage: requestedMessage, // "BootNotification", "StatusNotification", "Heartbeat", "MeterValues", etc.
		Evse:             evse,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "TriggerMessage", req)
	if err != nil {
		return nil, fmt.Errorf("trigger message failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("trigger message rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response TriggerMessageResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// --- Charging Profile Management ---

// SetChargingProfile sets a charging profile on an EVSE
func (s *Server) SetChargingProfile(ctx context.Context, chargePointID string, evseID int, profile ChargingProfile) (*SetChargingProfileResponse, error) {
	req := SetChargingProfileRequest{
		EvseId:          evseID,
		ChargingProfile: profile,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "SetChargingProfile", req)
	if err != nil {
		return nil, fmt.Errorf("set charging profile failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("set charging profile rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response SetChargingProfileResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// ClearChargingProfile clears charging profile(s) from a charge point
func (s *Server) ClearChargingProfile(ctx context.Context, chargePointID string, profileID *int, criteria *ClearChargingProfileCriteria) (*ClearChargingProfileResponse, error) {
	req := ClearChargingProfileRequest{
		ChargingProfileId:       profileID,
		ChargingProfileCriteria: criteria,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "ClearChargingProfile", req)
	if err != nil {
		return nil, fmt.Errorf("clear charging profile failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("clear charging profile rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response ClearChargingProfileResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetChargingProfiles requests charging profiles from a charge point
func (s *Server) GetChargingProfiles(ctx context.Context, chargePointID string, evseID *int, criteria *ChargingProfileCriterion) (*GetChargingProfilesResponse, error) {
	req := GetChargingProfilesRequest{
		RequestId:       int(time.Now().UnixNano() % 1000000),
		EvseId:          evseID,
		ChargingProfile: criteria,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "GetChargingProfiles", req)
	if err != nil {
		return nil, fmt.Errorf("get charging profiles failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("get charging profiles rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response GetChargingProfilesResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// --- Firmware Update ---

// UpdateFirmware requests a charge point to update its firmware
func (s *Server) UpdateFirmware(ctx context.Context, chargePointID, firmwareURL, retrieveDateTime string, installDateTime *string, retries, retryInterval *int) (*UpdateFirmwareResponse, error) {
	req := UpdateFirmwareRequest{
		RequestId: int(time.Now().UnixNano() % 1000000),
		Firmware: Firmware{
			Location:         firmwareURL,
			RetrieveDateTime: retrieveDateTime,
			InstallDateTime:  installDateTime,
		},
		Retries:       retries,
		RetryInterval: retryInterval,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "UpdateFirmware", req)
	if err != nil {
		return nil, fmt.Errorf("update firmware failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("update firmware rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response UpdateFirmwareResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateFirmwareSigned requests a signed firmware update (with certificate validation)
func (s *Server) UpdateFirmwareSigned(ctx context.Context, chargePointID string, firmwareURL, retrieveDateTime string, signingCert, signature string) (*UpdateFirmwareResponse, error) {
	req := UpdateFirmwareRequest{
		RequestId: int(time.Now().UnixNano() % 1000000),
		Firmware: Firmware{
			Location:           firmwareURL,
			RetrieveDateTime:   retrieveDateTime,
			SigningCertificate: &signingCert,
			Signature:          &signature,
		},
	}

	resp, err := s.SendCommand(ctx, chargePointID, "UpdateFirmware", req)
	if err != nil {
		return nil, fmt.Errorf("update firmware failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("update firmware rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response UpdateFirmwareResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// --- Variables ---

// GetVariables retrieves variable values from a charge point
func (s *Server) GetVariables(ctx context.Context, chargePointID string, variables []GetVariableData) (*GetVariablesResponse, error) {
	req := GetVariablesRequest{
		GetVariableData: variables,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "GetVariables", req)
	if err != nil {
		return nil, fmt.Errorf("get variables failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("get variables rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response GetVariablesResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// SetVariables sets variable values on a charge point
func (s *Server) SetVariables(ctx context.Context, chargePointID string, variables []SetVariableData) (*SetVariablesResponse, error) {
	req := SetVariablesRequest{
		SetVariableData: variables,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "SetVariables", req)
	if err != nil {
		return nil, fmt.Errorf("set variables failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("set variables rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response SetVariablesResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// --- Unlock Connector ---

// UnlockConnector requests to unlock a connector
func (s *Server) UnlockConnector(ctx context.Context, chargePointID string, evseID, connectorID int) (*UnlockConnectorResponse, error) {
	req := UnlockConnectorRequest{
		EvseId:      evseID,
		ConnectorId: connectorID,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "UnlockConnector", req)
	if err != nil {
		return nil, fmt.Errorf("unlock connector failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("unlock connector rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response UnlockConnectorResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// --- Change Availability ---

// ChangeAvailability changes the availability of a charge point or EVSE
func (s *Server) ChangeAvailability(ctx context.Context, chargePointID string, operationalStatus string, evse *Evse) (*ChangeAvailabilityResponse, error) {
	req := ChangeAvailabilityRequest{
		OperationalStatus: operationalStatus, // "Operative" or "Inoperative"
		Evse:              evse,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "ChangeAvailability", req)
	if err != nil {
		return nil, fmt.Errorf("change availability failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("change availability rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response ChangeAvailabilityResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// --- Diagnostics ---

// GetLog requests diagnostic logs from a charge point
func (s *Server) GetLog(ctx context.Context, chargePointID, logType, uploadURL string, oldestTimestamp, latestTimestamp *string, retries, retryInterval *int) (*GetLogResponse, error) {
	req := GetLogRequest{
		LogType:   logType, // "DiagnosticsLog" or "SecurityLog"
		RequestId: int(time.Now().UnixNano() % 1000000),
		Log: LogParams{
			RemoteLocation:  uploadURL,
			OldestTimestamp: oldestTimestamp,
			LatestTimestamp: latestTimestamp,
		},
		Retries:       retries,
		RetryInterval: retryInterval,
	}

	resp, err := s.SendCommand(ctx, chargePointID, "GetLog", req)
	if err != nil {
		return nil, fmt.Errorf("get log failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("get log rejected: %s - %s", resp.Error.Code, resp.Error.Description)
	}

	var response GetLogResponse
	if err := json.Unmarshal(resp.Payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// --- V2G Specific Commands ---

// SetV2GChargingProfile sets a bidirectional charging profile for V2G
func (s *Server) SetV2GChargingProfile(ctx context.Context, chargePointID string, evseID int, dischargePowerKW float64, duration int, minSOC int) (*SetChargingProfileResponse, error) {
	now := time.Now().Format(time.RFC3339)

	// Create a V2G charging profile with negative limits for discharge
	profile := ChargingProfile{
		Id:                     int(time.Now().UnixNano() % 1000000),
		StackLevel:             0, // Highest priority
		ChargingProfilePurpose: "TxProfile",
		ChargingProfileKind:    "Absolute",
		ValidFrom:              &now,
		ChargingSchedule: []ChargingSchedule{
			{
				Id:               1,
				ChargingRateUnit: "W",
				Duration:         &duration,
				ChargingSchedulePeriod: []ChargingSchedulePeriod{
					{
						StartPeriod: 0,
						Limit:       -dischargePowerKW * 1000, // Negative for discharge, convert kW to W
					},
				},
			},
		},
	}

	return s.SetChargingProfile(ctx, chargePointID, evseID, profile)
}

// CancelV2GDischarge stops V2G discharge by clearing the profile
func (s *Server) CancelV2GDischarge(ctx context.Context, chargePointID string, evseID int) (*ClearChargingProfileResponse, error) {
	criteria := &ClearChargingProfileCriteria{
		EvseId:                 &evseID,
		ChargingProfilePurpose: stringPtr("TxProfile"),
	}
	return s.ClearChargingProfile(ctx, chargePointID, nil, criteria)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
