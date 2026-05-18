package ingestion

import "context"

// Extractor normaliza contenido crudo referenciado por StorageRef en uno o más artifacts.
// Implementaciones viven en workers HTTP, Lambdas dedicadas o bibliotecas locales.
type Extractor interface {
	Extract(ctx context.Context, req ExtractRequest) (*ExtractResponse, error)
}
