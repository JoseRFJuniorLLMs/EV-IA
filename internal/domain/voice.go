package domain

type VoiceResponse struct {
	Text   string `json:"text"`
	Audio  string `json:"audio,omitempty"` // Base64 encoded audio
	Intent string `json:"intent,omitempty"`
}

type Intent struct {
	Name       string                 `json:"name"`
	Confidence float64                `json:"confidence"`
	Slots      map[string]interface{} `json:"slots,omitempty"`
}
