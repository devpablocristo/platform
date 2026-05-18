package ginmw

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ParseAfterUUIDQuery reads the `after` cursor as a UUID. Returns (nil, true)
// when the param is absent or empty. Returns (nil, false) and writes a 400
// `validation` error when the value is present but malformed.
func ParseAfterUUIDQuery(c *gin.Context) (*uuid.UUID, bool) {
	v := strings.TrimSpace(c.Query("after"))
	if v == "" {
		return nil, true
	}
	id, err := uuid.Parse(v)
	if err != nil {
		WriteValidation(c, "invalid after")
		return nil, false
	}
	return &id, true
}

// WriteListResponse writes a JSON 200 with the conventional CRUDAR list shape:
//
//	{
//	  "items":       [...],
//	  "total":       <int64>,
//	  "has_more":    <bool>,
//	  "next_cursor": "<string>"
//	}
//
// Pagination semantics (cursor vs offset, total vs has_more) are decided by
// the caller. This helper only formats the response.
func WriteListResponse(c *gin.Context, items any, total int64, hasMore bool, nextCursor string) {
	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"total":       total,
		"has_more":    hasMore,
		"next_cursor": nextCursor,
	})
}

// WriteOffsetListResponse is the offset variant: `total` is the global count
// in the table, and `has_more` is derived from total > limit. Cursor is left
// empty (offset pagination does not use it).
func WriteOffsetListResponse(c *gin.Context, items any, limit int, total int) {
	WriteListResponse(c, items, int64(total), total > limit, "")
}

// WriteValidation writes a 400 with the standardized envelope:
//
//	{"code": "VALIDATION", "message": "<msg>"}
//
// Use this when the request body / params fail consumer-side validation that
// is *not* expressible as domainerr (where ginmw.Respond is preferred).
func WriteValidation(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION", "message": message})
}
