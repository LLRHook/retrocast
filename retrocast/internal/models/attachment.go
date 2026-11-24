package models

// Attachment represents a file attached to a message.
type Attachment struct {
	ID          int64  `json:"id,string"`
	MessageID   *int64 `json:"message_id,string"`
	UploaderID  int64  `json:"-"`
	ChannelID   int64  `json:"-"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	StorageKey  string `json:"-"`
	URL         string `json:"url"`
}
