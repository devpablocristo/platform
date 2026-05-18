package ingestion

// Códigos de error estables para mapear a estados de job en el producto (sin filtrar internals al cliente HTTP público).
const (
	CodeUnsupported   = "EXTRACTION_UNSUPPORTED"
	CodeTimeout       = "EXTRACTION_TIMEOUT"
	CodeUpstream      = "EXTRACTION_UPSTREAM_ERROR"
	CodeValidation    = "EXTRACTION_VALIDATION"
	CodeUnauthorized  = "EXTRACTION_UNAUTHORIZED"
	CodeInternal      = "EXTRACTION_INTERNAL"
)
