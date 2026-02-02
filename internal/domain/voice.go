package domain

type VoiceResponse struct {
	Text         string  `json:"text"`
	Audio        []byte  `json:"audio,omitempty"` // PCM audio bytes
	Intent       string  `json:"intent,omitempty"`
	ActionResult string  `json:"action_result,omitempty"`
	Confidence   float64 `json:"confidence,omitempty"`
}

type Intent struct {
	Name       string            `json:"name"`
	Confidence float64           `json:"confidence"`
	Entities   map[string]string `json:"entities,omitempty"`
	Slots      map[string]interface{} `json:"slots,omitempty"`
}
