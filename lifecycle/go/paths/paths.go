// Package paths defines the HTTP route segments for the canonical CRUDAR
// contract (archived list, restore, hard delete).
//
// These literals match the TS counterpart in @devpablocristo/platform-crud-ui
// (src/restPaths.ts). They are kept here under platform/lifecycle/go/paths
// as the canonical home; the legacy github.com/devpablocristo/platform/features/crud/paths/go
// package re-exports the same constants and will be removed in a follow-up.
package paths

const (
	SegmentArchived = "archived"
	SegmentArchive  = "archive"
	SegmentRestore  = "restore"
	SegmentHard     = "hard"
)
