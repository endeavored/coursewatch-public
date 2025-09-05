package models

type SlackAccessory struct {
	Type     string `json:"type"`
	ImageUrl string `json:"image_url"`
	AltText  string `json:"alt_text"`
}
type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type SlackBlock struct {
	Type      string          `json:"type"`
	Text      *SlackText      `json:"text,omitempty"`
	Accessory *SlackAccessory `json:"accessory,omitempty"`
	BlockId   string          `json:"block_id,omitempty"`
	Fields    *[]SlackText    `json:"fields,omitempty"`
}
type SlackDirectMessageData struct {
	Token   string       `json:"token"`
	Channel string       `json:"channel"`
	Blocks  []SlackBlock `json:"blocks"`
}
type SlackWebhookData struct {
	Text   string       `json:"text,omitempty"`
	Blocks []SlackBlock `json:"blocks"`
}
type SlackSocketData struct {
	EnvelopeId             string             `json:"envelope_id"`
	Payload                SlackSocketPayload `json:"payload"`
	Type                   string             `json:"type"`
	AcceptsResponsePayload bool               `json:"accepts_response_payload"`
}
type SlackSocketPayload struct {
	Token        string `json:"token"`
	TeamId       string `json:"team_id"`
	TeamDomain   string `json:"team_domain"`
	ChannelId    string `json:"channel_id"`
	ChannelName  string `json:"channel_name"`
	UserId       string `json:"user_id"`
	UserName     string `json:"user_name"`
	Command      string `json:"command"`
	Text         string `json:"text"`
	ApiAppId     string `json:"api_app_id"`
	IsEnterprise string `json:"is_enterprise_install"`
	ResponseUrl  string `json:"response_url"`
	TriggerId    string `json:"trigger_id"`
}
