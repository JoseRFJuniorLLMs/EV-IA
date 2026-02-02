package v201

// MessageType represents the type of OCPP message
type MessageType int

const (
	Call       MessageType = 2
	CallResult MessageType = 3
	CallError  MessageType = 4
)

// CallMessage represents a request message
type CallMessage struct {
	MessageTypeID MessageType `json:"messageTypeId"`
	MessageID     string      `json:"messageId"`
	Action        string      `json:"action"`
	Payload       interface{} `json:"payload"`
}

// CallResultMessage represents a response message
type CallResultMessage struct {
	MessageTypeID MessageType `json:"messageTypeId"`
	MessageID     string      `json:"messageId"`
	Payload       interface{} `json:"payload"`
}

// CallErrorMessage represents an error message
type CallErrorMessage struct {
	MessageTypeID    MessageType `json:"messageTypeId"`
	MessageID        string      `json:"messageId"`
	ErrorCode        string      `json:"errorCode"`
	ErrorDescription string      `json:"errorDescription"`
	ErrorDetails     interface{} `json:"errorDetails"`
}

// --- Payload Structs ---

type BootNotificationRequest struct {
	ChargingStation ChargingStation `json:"chargingStation"`
	Reason          string          `json:"reason"`
}

type ChargingStation struct {
	Model           string `json:"model"`
	VendorName      string `json:"vendorName"`
	SerialNumber    string `json:"serialNumber,omitempty"`
	FirmwareVersion string `json:"firmwareVersion,omitempty"`
}

type BootNotificationResponse struct {
	CurrentTime string `json:"currentTime"`
	Interval    int    `json:"interval"`
	Status      string `json:"status"` // Accepted, Pending, Rejected
}

type HeartbeatRequest struct{}

type HeartbeatResponse struct {
	CurrentTime string `json:"currentTime"`
}

type StatusNotificationRequest struct {
	Timestamp       string `json:"timestamp"`
	ConnectorStatus string `json:"connectorStatus"` // Available, Occupied, Reserved, Unavailable, Faulted
	EvseId          int    `json:"evseId"`
	ConnectorId     int    `json:"connectorId"`
}

type StatusNotificationResponse struct{}

type TransactionEventRequest struct {
	EventType       string          `json:"eventType"` // Started, Updated, Ended
	Timestamp       string          `json:"timestamp"`
	TriggerReason   string          `json:"triggerReason"`
	SeqNo           int             `json:"seqNo"`
	TransactionInfo TransactionInfo `json:"transactionInfo"`
	IdToken         *IdToken        `json:"idToken,omitempty"`
	Evse            *Evse           `json:"evse,omitempty"`
	MeterValue      []MeterValue    `json:"meterValue,omitempty"`
}

type TransactionInfo struct {
	TransactionId string `json:"transactionId"`
}

type IdToken struct {
	IdToken string `json:"idToken"`
	Type    string `json:"type"` // Isotill 14443, MacAddress, etc
}

type Evse struct {
	Id          int `json:"id"`
	ConnectorId int `json:"connectorId"`
}

type MeterValue struct {
	Timestamp    string         `json:"timestamp"`
	SampledValue []SampledValue `json:"sampledValue"`
}

type SampledValue struct {
	Value     string `json:"value"`
	Context   string `json:"context,omitempty"`
	Measurand string `json:"measurand,omitempty"` // Energy.Active.Import.Register
	Unit      string `json:"unit,omitempty"`      // Wh, kWh
}

type TransactionEventResponse struct {
	IdTokenInfo *IdTokenInfo `json:"idTokenInfo,omitempty"`
}

type IdTokenInfo struct {
	Status string `json:"status"` // Accepted, Blocked, Expired, Invalid, ConcurrentTx
}

// StatusInfo provides additional status information
type StatusInfo struct {
	ReasonCode     string `json:"reasonCode"`
	AdditionalInfo string `json:"additionalInfo,omitempty"`
}

// --- CSMS → Charge Point Messages ---

// RequestStartTransactionRequest - CSMS requests charge point to start a transaction
type RequestStartTransactionRequest struct {
	IdToken         IdToken          `json:"idToken"`
	RemoteStartId   int              `json:"remoteStartId"`
	EvseId          *int             `json:"evseId,omitempty"`
	ChargingProfile *ChargingProfile `json:"chargingProfile,omitempty"`
}

// RequestStartTransactionResponse - Response from charge point
type RequestStartTransactionResponse struct {
	Status        string      `json:"status"` // Accepted, Rejected
	TransactionId string      `json:"transactionId,omitempty"`
	StatusInfo    *StatusInfo `json:"statusInfo,omitempty"`
}

// RequestStopTransactionRequest - CSMS requests charge point to stop a transaction
type RequestStopTransactionRequest struct {
	TransactionId string `json:"transactionId"`
}

// RequestStopTransactionResponse - Response from charge point
type RequestStopTransactionResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// ChargingProfile defines a charging schedule
type ChargingProfile struct {
	Id                     int                `json:"id"`
	StackLevel             int                `json:"stackLevel"`
	ChargingProfilePurpose string             `json:"chargingProfilePurpose"` // ChargePointMaxProfile, TxDefaultProfile, TxProfile
	ChargingProfileKind    string             `json:"chargingProfileKind"`    // Absolute, Recurring, Relative
	RecurrencyKind         string             `json:"recurrencyKind,omitempty"`
	ValidFrom              *string            `json:"validFrom,omitempty"`
	ValidTo                *string            `json:"validTo,omitempty"`
	ChargingSchedule       []ChargingSchedule `json:"chargingSchedule"`
}

// ChargingSchedule defines the charging schedule periods
type ChargingSchedule struct {
	Id                     int                       `json:"id"`
	StartSchedule          *string                   `json:"startSchedule,omitempty"`
	Duration               *int                      `json:"duration,omitempty"`
	ChargingRateUnit       string                    `json:"chargingRateUnit"` // W, A
	MinChargingRate        *float64                  `json:"minChargingRate,omitempty"`
	ChargingSchedulePeriod []ChargingSchedulePeriod  `json:"chargingSchedulePeriod"`
	SalesTariff            *SalesTariff              `json:"salesTariff,omitempty"`
}

// ChargingSchedulePeriod defines a period within a charging schedule
type ChargingSchedulePeriod struct {
	StartPeriod  int      `json:"startPeriod"` // Seconds from start
	Limit        float64  `json:"limit"`       // Power limit (positive = charge, negative = discharge for V2G)
	NumberPhases *int     `json:"numberPhases,omitempty"`
	PhaseToUse   *int     `json:"phaseToUse,omitempty"`
}

// SalesTariff for pricing information
type SalesTariff struct {
	Id                     int                 `json:"id"`
	SalesTariffDescription string              `json:"salesTariffDescription,omitempty"`
	NumEPriceLevels        *int                `json:"numEPriceLevels,omitempty"`
	SalesTariffEntry       []SalesTariffEntry  `json:"salesTariffEntry"`
}

// SalesTariffEntry defines pricing for a period
type SalesTariffEntry struct {
	RelativeTimeInterval RelativeTimeInterval `json:"relativeTimeInterval"`
	EPriceLevel          *int                 `json:"ePriceLevel,omitempty"`
	ConsumptionCost      []ConsumptionCost    `json:"consumptionCost,omitempty"`
}

// RelativeTimeInterval for tariff periods
type RelativeTimeInterval struct {
	Start    int  `json:"start"`
	Duration *int `json:"duration,omitempty"`
}

// ConsumptionCost for detailed pricing
type ConsumptionCost struct {
	StartValue float64 `json:"startValue"`
	Cost       []Cost  `json:"cost"`
}

// Cost structure for pricing
type Cost struct {
	CostKind string  `json:"costKind"` // CarbonDioxideEmission, RelativePricePercentage, RenewableGenerationPercentage
	Amount   float64 `json:"amount"`
}

// SetChargingProfileRequest - CSMS sets a charging profile on the charge point
type SetChargingProfileRequest struct {
	EvseId          int             `json:"evseId"`
	ChargingProfile ChargingProfile `json:"chargingProfile"`
}

// SetChargingProfileResponse - Response from charge point
type SetChargingProfileResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// ClearChargingProfileRequest - CSMS clears charging profile(s)
type ClearChargingProfileRequest struct {
	ChargingProfileId       *int                         `json:"chargingProfileId,omitempty"`
	ChargingProfileCriteria *ClearChargingProfileCriteria `json:"chargingProfileCriteria,omitempty"`
}

// ClearChargingProfileCriteria defines which profiles to clear
type ClearChargingProfileCriteria struct {
	EvseId                 *int    `json:"evseId,omitempty"`
	ChargingProfilePurpose *string `json:"chargingProfilePurpose,omitempty"`
	StackLevel             *int    `json:"stackLevel,omitempty"`
}

// ClearChargingProfileResponse - Response from charge point
type ClearChargingProfileResponse struct {
	Status     string      `json:"status"` // Accepted, Unknown
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// GetChargingProfilesRequest - CSMS requests charging profiles
type GetChargingProfilesRequest struct {
	RequestId              int                           `json:"requestId"`
	EvseId                 *int                          `json:"evseId,omitempty"`
	ChargingProfile        *ChargingProfileCriterion     `json:"chargingProfile,omitempty"`
}

// ChargingProfileCriterion for filtering profiles
type ChargingProfileCriterion struct {
	ChargingProfilePurpose *string `json:"chargingProfilePurpose,omitempty"`
	StackLevel             *int    `json:"stackLevel,omitempty"`
	ChargingProfileId      []int   `json:"chargingProfileId,omitempty"`
	ChargingLimitSource    []string `json:"chargingLimitSource,omitempty"`
}

// GetChargingProfilesResponse - Response from charge point
type GetChargingProfilesResponse struct {
	Status     string      `json:"status"` // Accepted, NoProfiles
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// ReportChargingProfilesRequest - Charge point reports profiles (async response)
type ReportChargingProfilesRequest struct {
	RequestId          int               `json:"requestId"`
	ChargingLimitSource string           `json:"chargingLimitSource"`
	EvseId             int               `json:"evseId"`
	ChargingProfile    []ChargingProfile `json:"chargingProfile"`
	Tbc                bool              `json:"tbc,omitempty"` // To be continued
}

// ReportChargingProfilesResponse - CSMS acknowledges
type ReportChargingProfilesResponse struct{}

// ResetRequest - CSMS requests charge point to reset
type ResetRequest struct {
	Type   string `json:"type"` // Immediate, OnIdle
	EvseId *int   `json:"evseId,omitempty"`
}

// ResetResponse - Response from charge point
type ResetResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, Scheduled
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// UpdateFirmwareRequest - CSMS requests firmware update
type UpdateFirmwareRequest struct {
	RequestId     int      `json:"requestId"`
	Firmware      Firmware `json:"firmware"`
	Retries       *int     `json:"retries,omitempty"`
	RetryInterval *int     `json:"retryInterval,omitempty"`
}

// Firmware details for update
type Firmware struct {
	Location           string  `json:"location"` // URL to download firmware
	RetrieveDateTime   string  `json:"retrieveDateTime"`
	InstallDateTime    *string `json:"installDateTime,omitempty"`
	SigningCertificate *string `json:"signingCertificate,omitempty"`
	Signature          *string `json:"signature,omitempty"`
}

// UpdateFirmwareResponse - Response from charge point
type UpdateFirmwareResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, AcceptedCanceled, InvalidCertificate, RevokedCertificate
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// FirmwareStatusNotificationRequest - Charge point notifies firmware status
type FirmwareStatusNotificationRequest struct {
	Status    string `json:"status"` // Downloaded, DownloadFailed, Downloading, DownloadScheduled, DownloadPaused, Idle, InstallationFailed, Installing, Installed, InstallRebooting, InstallScheduled, InstallVerificationFailed, InvalidSignature, SignatureVerified
	RequestId *int   `json:"requestId,omitempty"`
}

// FirmwareStatusNotificationResponse - CSMS acknowledges
type FirmwareStatusNotificationResponse struct{}

// GetVariablesRequest - CSMS requests variable values
type GetVariablesRequest struct {
	GetVariableData []GetVariableData `json:"getVariableData"`
}

// GetVariableData specifies which variable to get
type GetVariableData struct {
	Component       Component  `json:"component"`
	Variable        Variable   `json:"variable"`
	AttributeType   string     `json:"attributeType,omitempty"` // Actual, Target, MinSet, MaxSet
}

// Component identifies a component
type Component struct {
	Name     string    `json:"name"`
	Instance string    `json:"instance,omitempty"`
	Evse     *Evse     `json:"evse,omitempty"`
}

// Variable identifies a variable
type Variable struct {
	Name     string `json:"name"`
	Instance string `json:"instance,omitempty"`
}

// GetVariablesResponse - Response with variable values
type GetVariablesResponse struct {
	GetVariableResult []GetVariableResult `json:"getVariableResult"`
}

// GetVariableResult contains a variable's value
type GetVariableResult struct {
	AttributeStatus string       `json:"attributeStatus"` // Accepted, Rejected, UnknownComponent, UnknownVariable, NotSupportedAttributeType
	AttributeType   string       `json:"attributeType,omitempty"`
	AttributeValue  string       `json:"attributeValue,omitempty"`
	Component       Component    `json:"component"`
	Variable        Variable     `json:"variable"`
	StatusInfo      *StatusInfo  `json:"statusInfo,omitempty"`
}

// SetVariablesRequest - CSMS sets variable values
type SetVariablesRequest struct {
	SetVariableData []SetVariableData `json:"setVariableData"`
}

// SetVariableData specifies variable to set
type SetVariableData struct {
	AttributeType  string    `json:"attributeType,omitempty"`
	AttributeValue string    `json:"attributeValue"`
	Component      Component `json:"component"`
	Variable       Variable  `json:"variable"`
}

// SetVariablesResponse - Response from charge point
type SetVariablesResponse struct {
	SetVariableResult []SetVariableResult `json:"setVariableResult"`
}

// SetVariableResult contains result for each variable
type SetVariableResult struct {
	AttributeStatus string      `json:"attributeStatus"` // Accepted, Rejected, UnknownComponent, UnknownVariable, NotSupportedAttributeType, RebootRequired
	Component       Component   `json:"component"`
	Variable        Variable    `json:"variable"`
	AttributeType   string      `json:"attributeType,omitempty"`
	StatusInfo      *StatusInfo `json:"statusInfo,omitempty"`
}

// TriggerMessageRequest - CSMS triggers a message from charge point
type TriggerMessageRequest struct {
	RequestedMessage string `json:"requestedMessage"` // BootNotification, LogStatusNotification, FirmwareStatusNotification, Heartbeat, MeterValues, etc.
	Evse             *Evse  `json:"evse,omitempty"`
}

// TriggerMessageResponse - Response from charge point
type TriggerMessageResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, NotImplemented
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// UnlockConnectorRequest - CSMS requests connector unlock
type UnlockConnectorRequest struct {
	EvseId      int `json:"evseId"`
	ConnectorId int `json:"connectorId"`
}

// UnlockConnectorResponse - Response from charge point
type UnlockConnectorResponse struct {
	Status     string      `json:"status"` // Unlocked, UnlockFailed, OngoingAuthorizedTransaction, UnknownConnector
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// ChangeAvailabilityRequest - CSMS changes charge point/EVSE availability
type ChangeAvailabilityRequest struct {
	OperationalStatus string `json:"operationalStatus"` // Operative, Inoperative
	Evse              *Evse  `json:"evse,omitempty"`
}

// ChangeAvailabilityResponse - Response from charge point
type ChangeAvailabilityResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, Scheduled
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// --- V2G (Vehicle-to-Grid) Messages ---

// NotifyEVChargingNeedsRequest - EV notifies charging/discharging needs
type NotifyEVChargingNeedsRequest struct {
	EvseId            int           `json:"evseId"`
	ChargingNeeds     ChargingNeeds `json:"chargingNeeds"`
	MaxScheduleTuples *int          `json:"maxScheduleTuples,omitempty"`
}

// ChargingNeeds defines EV charging requirements
type ChargingNeeds struct {
	RequestedEnergyTransfer string                `json:"requestedEnergyTransfer"` // AC_single_phase, AC_two_phase, AC_three_phase, DC, AC_BPT, DC_BPT (BPT = Bidirectional Power Transfer for V2G)
	DepartureTime           *string               `json:"departureTime,omitempty"`
	ACChargingParameters    *ACChargingParameters `json:"acChargingParameters,omitempty"`
	DCChargingParameters    *DCChargingParameters `json:"dcChargingParameters,omitempty"`
}

// ACChargingParameters for AC charging
type ACChargingParameters struct {
	EnergyAmount int `json:"energyAmount"` // Wh
	EVMinCurrent int `json:"evMinCurrent"` // A
	EVMaxCurrent int `json:"evMaxCurrent"` // A
	EVMaxVoltage int `json:"evMaxVoltage"` // V
}

// DCChargingParameters for DC charging (including V2G)
type DCChargingParameters struct {
	EVMaxCurrent          int  `json:"evMaxCurrent"`                    // A
	EVMaxVoltage          int  `json:"evMaxVoltage"`                    // V
	FullSOC               *int `json:"fullSoc,omitempty"`               // Target SOC %
	BulkSOC               *int `json:"bulkSoc,omitempty"`               // Bulk charge SOC %
	StateOfCharge         int  `json:"stateOfCharge"`                   // Current battery SOC %
	EVEnergyCapacity      *int `json:"evEnergyCapacity,omitempty"`      // kWh - Battery capacity
	EVMaxDischargePower   *int `json:"evMaxDischargePower,omitempty"`   // W - Max V2G discharge power
	EVMaxDischargeCurrent *int `json:"evMaxDischargeCurrent,omitempty"` // A - Max V2G discharge current
}

// NotifyEVChargingNeedsResponse - CSMS acknowledges
type NotifyEVChargingNeedsResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected, Processing
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// NotifyEVChargingScheduleRequest - Charge point notifies scheduled charging
type NotifyEVChargingScheduleRequest struct {
	TimeBase         string           `json:"timeBase"`
	EvseId           int              `json:"evseId"`
	ChargingSchedule ChargingSchedule `json:"chargingSchedule"`
}

// NotifyEVChargingScheduleResponse - CSMS acknowledges
type NotifyEVChargingScheduleResponse struct {
	Status     string      `json:"status"` // Accepted, Rejected
	StatusInfo *StatusInfo `json:"statusInfo,omitempty"`
}

// --- ISO 15118 Messages (Plug & Charge) ---

// Get15118EVCertificateRequest - EV requests certificate
type Get15118EVCertificateRequest struct {
	ISO15118SchemaVersion string `json:"iso15118SchemaVersion"`
	Action                string `json:"action"` // Install, Update
	EXIRequest            string `json:"exiRequest"`
}

// Get15118EVCertificateResponse - Response with certificate
type Get15118EVCertificateResponse struct {
	Status      string      `json:"status"` // Accepted, Failed
	EXIResponse string      `json:"exiResponse"`
	StatusInfo  *StatusInfo `json:"statusInfo,omitempty"`
}

// Authorize15118Request - Authorization via ISO 15118
type Authorize15118Request struct {
	IdToken           IdToken       `json:"idToken"`
	Certificate       *string       `json:"certificate,omitempty"`
	ISO15118CertificateHashData []OCSPRequestData `json:"iso15118CertificateHashData,omitempty"`
}

// OCSPRequestData for certificate validation
type OCSPRequestData struct {
	HashAlgorithm  string `json:"hashAlgorithm"` // SHA256, SHA384, SHA512
	IssuerNameHash string `json:"issuerNameHash"`
	IssuerKeyHash  string `json:"issuerKeyHash"`
	SerialNumber   string `json:"serialNumber"`
	ResponderURL   string `json:"responderURL,omitempty"`
}

// AuthorizeResponse - Authorization response
type AuthorizeResponse struct {
	IdTokenInfo          IdTokenInfo   `json:"idTokenInfo"`
	CertificateStatus    *string       `json:"certificateStatus,omitempty"` // Accepted, SignatureError, CertificateExpired, CertificateRevoked, NoCertificateAvailable, CertChainError, ContractCancelled
}

// --- Metering Messages ---

// MeterValuesRequest - Charge point sends meter values
type MeterValuesRequest struct {
	EvseId     int          `json:"evseId"`
	MeterValue []MeterValue `json:"meterValue"`
}

// MeterValuesResponse - CSMS acknowledges
type MeterValuesResponse struct{}

// --- Diagnostics Messages ---

// GetLogRequest - CSMS requests diagnostic log
type GetLogRequest struct {
	LogType       string    `json:"logType"` // DiagnosticsLog, SecurityLog
	RequestId     int       `json:"requestId"`
	Log           LogParams `json:"log"`
	Retries       *int      `json:"retries,omitempty"`
	RetryInterval *int      `json:"retryInterval,omitempty"`
}

// LogParams for log retrieval
type LogParams struct {
	RemoteLocation string  `json:"remoteLocation"` // URL to upload logs
	OldestTimestamp *string `json:"oldestTimestamp,omitempty"`
	LatestTimestamp *string `json:"latestTimestamp,omitempty"`
}

// GetLogResponse - Response from charge point
type GetLogResponse struct {
	Status   string `json:"status"` // Accepted, Rejected, AcceptedCanceled
	Filename string `json:"filename,omitempty"`
}

// LogStatusNotificationRequest - Charge point notifies log upload status
type LogStatusNotificationRequest struct {
	Status    string `json:"status"` // BadMessage, Idle, NotSupportedOperation, PermissionDenied, Uploaded, UploadFailure, Uploading, AcceptedCanceled
	RequestId *int   `json:"requestId,omitempty"`
}

// LogStatusNotificationResponse - CSMS acknowledges
type LogStatusNotificationResponse struct{}
