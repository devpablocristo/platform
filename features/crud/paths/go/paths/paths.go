// Package paths define segmentos de ruta HTTP para CRUD canónico.
// No registra rutas ni contiene dominio: mismos literales que usa el TS en platform/features/crud/ui/ts/src/restPaths.ts.
package paths

const (
	SegmentArchived  = "archived"
	SegmentArchive   = "archive"
	SegmentUnarchive = "unarchive"
	SegmentTrash     = "trash"
	SegmentRestore   = "restore"
	SegmentPurge     = "purge"
)
