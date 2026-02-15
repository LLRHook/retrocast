package models

type Attachment struct {
	ID          int64  `json:"id,string"`
	MessageID   int64  `json:"message_id,string"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	StorageKey  string `json:"storage_key"`
}
