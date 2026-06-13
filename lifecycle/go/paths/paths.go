// Package paths defines the HTTP route segments for the canonical CRUDAR
// contract.
//
// These literals match the TS counterpart in @devpablocristo/platform-crud-ui
// (src/restPaths.ts). They are kept here under platform/lifecycle/go/paths
// as the canonical home.
package paths

const (
	SegmentArchived  = "archived"
	SegmentArchive   = "archive"
	SegmentUnarchive = "unarchive"
	SegmentTrash     = "trash"
	SegmentRestore   = "restore"
	SegmentPurge     = "purge"
)
