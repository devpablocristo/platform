// Package paths define segmentos de ruta HTTP para CRUD canónico (listado archivado, restore, hard delete).
// No registra rutas ni contiene dominio: mismos literales que usa el TS en modules/crud/ui/ts/src/restPaths.ts.
package paths

const (
	SegmentArchived = "archived"
	SegmentArchive  = "archive"
	SegmentRestore  = "restore"
	SegmentHard     = "hard"
)
