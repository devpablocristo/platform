package attachments

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	domain "github.com/devpablocristo/core/artifact/go/attachments/usecases/domain"
)

func BuildStorageKey(tenantID, attachableType, attachableID, fileName string) (string, error) {
	tenantID = strings.TrimSpace(tenantID)
	attachableType = sanitizeSegment(attachableType)
	attachableID = strings.TrimSpace(attachableID)
	if tenantID == "" || attachableType == "" || attachableID == "" {
		return "", fmt.Errorf("tenant_id, attachable_type and attachable_id are required")
	}
	return filepath.ToSlash(filepath.Join(
		tenantID,
		attachableType,
		attachableID,
		fmt.Sprintf("%d-%s", time.Now().UTC().UnixNano(), SanitizeFileName(fileName)),
	)), nil
}

func RequestUpload(baseURL, storageKey string, ttl time.Duration) (domain.UploadRequest, error) {
	if strings.TrimSpace(storageKey) == "" {
		return domain.UploadRequest{}, fmt.Errorf("storage_key is required")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	now := time.Now().UTC()
	return domain.UploadRequest{
		StorageKey: storageKey,
		UploadURL:  baseURL + "/uploads/" + storageKey,
		ExpiresAt:  now.Add(ttl),
	}, nil
}

func BuildDownloadLink(baseURL, id string, ttl time.Duration) (domain.DownloadLink, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.DownloadLink{}, fmt.Errorf("attachment id is required")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	now := time.Now().UTC()
	return domain.DownloadLink{
		URL:       baseURL + "/attachments/" + id + "/download",
		ExpiresAt: now.Add(ttl),
	}, nil
}

func SanitizeFileName(value string) string {
	name := strings.TrimSpace(value)
	if name == "" {
		return "file.bin"
	}
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	return name
}

func sanitizeSegment(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "\\", "-")
	return strings.ReplaceAll(value, " ", "-")
}
