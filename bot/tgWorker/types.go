package telegram

type UpdatesRequest struct {
	UpdateID int64     `json:"update_id"`
	Message  TGMessage `json:"message"`
}

type TGMessage struct {
	MessageID int64  `json:"message_id"`
	Text      string `json:"text"`
	Chat      TGChat `json:"chat"`
}

type TGChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}
