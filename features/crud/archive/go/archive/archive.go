// Package archive provee helpers para el estado de archivado (soft delete)
// del contrato CRUD canónico (ver github.com/devpablocristo/modules/crud/paths).
//
// ErrArchived es un domainerr.Error con KindConflict: los middlewares HTTP
// estándar (ej. core/http/gin/go) lo mapean automáticamente a 409 Conflict.
package archive

import (
	"fmt"
	"time"

	"github.com/devpablocristo/core/errors/go/domainerr"
)

// ErrArchived indica que la operación no se puede ejecutar porque el recurso
// está archivado (soft-deleted). Como es domainerr.Conflict, los routers HTTP
// lo mapean a 409.
//
// Uso típico:
//
//	if err := archive.IfArchived(current.DeletedAt, "product"); err != nil {
//	    return err
//	}
var ErrArchived = domainerr.Conflict("resource is archived")

// IfArchived retorna ErrArchived envuelto si archivedAt no es nil.
// El prefijo resource se incluye en el mensaje para trazabilidad.
//
// El puntero archivedAt sigue la convención GORM/SQL: nil ⇒ activo,
// valor ⇒ archivado en ese timestamp.
func IfArchived(archivedAt *time.Time, resource string) error {
	if archivedAt == nil {
		return nil
	}
	return fmt.Errorf("%s archived: %w", resource, ErrArchived)
}

// IsArchived es el predicado puro equivalente a archivedAt != nil.
// Útil cuando sólo se necesita saber el estado sin propagar error.
func IsArchived(archivedAt *time.Time) bool {
	return archivedAt != nil
}
