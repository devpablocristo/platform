package domain

import "time"

type Attachment struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	AttachableType string    `json:"attachable_type"`
	AttachableID   string    `json:"attachable_id"`
	FileName       string    `json:"file_name"`
	ContentType    string    `json:"content_type"`
	SizeBytes      int64     `json:"size_bytes"`
	StorageKey     string    `json:"storage_key"`
	UploadedBy     string    `json:"uploaded_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type UploadRequest struct {
	StorageKey string    `json:"storage_key"`
	UploadURL  string    `json:"upload_url"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type DownloadLink struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}
