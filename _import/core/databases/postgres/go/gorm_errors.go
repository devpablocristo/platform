package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// HandleNotFound convierte gorm.ErrRecordNotFound a un error con formato estándar.
// Para otros errores retorna un error genérico sin exponer detalles internos.
func HandleNotFound(err error, entity string, id any) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%s %v not found", entity, id)
	}
	return fmt.Errorf("failed to get %s", entity)
}

// IsNotFound retorna true si el error es gorm.ErrRecordNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// IsUniqueViolation detecta violaciones de constraint UNIQUE de PostgreSQL y
// errores equivalentes normalizados por GORM.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23505"
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "23505") ||
		strings.Contains(msg, "unique") ||
		strings.Contains(msg, "duplicate")
}

type txContextKey struct{}

// WithTx adjunta una transacción GORM al contexto para que los repositorios la reutilicen.
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// TxFromContext extrae la transacción del contexto si existe.
func TxFromContext(ctx context.Context) *gorm.DB {
	tx, _ := ctx.Value(txContextKey{}).(*gorm.DB)
	return tx
}

// DBOrTx retorna la transacción del contexto si existe, o el db base.
func DBOrTx(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return db
}
