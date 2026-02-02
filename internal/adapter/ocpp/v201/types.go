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
