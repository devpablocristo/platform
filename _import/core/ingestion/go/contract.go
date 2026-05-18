package ingestion

// StorageRef apunta al binario original en almacenamiento (p. ej. S3).
// El servicio de extracción usa Kind + Key (+ Bucket si aplica) con credenciales propias.
type StorageRef struct {
	Kind   string `json:"kind"`             // ej: "s3"
	Bucket string `json:"bucket,omitempty"` // opcional si el bucket es implícito en el worker
	Key    string `json:"key"`
}

// ExtractRequest es la entrada estable del contrato de extracción/normalización (agnóstico al dominio clínico).
type ExtractRequest struct {
	AssetID      string     `json:"assetId"`
	ContentType  string     `json:"contentType"`
	ByteSize     int64      `json:"byteSize"`
	StorageRef   StorageRef `json:"storageRef"`
	LocaleHint   string     `json:"localeHint,omitempty"`
	CorrelationID string    `json:"correlationId,omitempty"` // ej: jobId para trazas
}

// Provenance identifica motor y versión de un paso de extracción.
type Provenance struct {
	Engine        string `json:"engine"`
	EngineVersion string `json:"engineVersion,omitempty"`
	Step          string `json:"step,omitempty"`
}

// NormalizedArtifact es una salida canónica por paso/motor (texto o payload estructurado).
type NormalizedArtifact struct {
	Kind        string         `json:"kind"` // ej: text_document, audio_transcript, tabular_document
	Version     int            `json:"version"`
	FullText    string         `json:"fullText,omitempty"`
	Structured  map[string]any `json:"structured,omitempty"`
	Provenance  Provenance     `json:"provenance"`
	Confidence  *float64      `json:"confidence,omitempty"`
}

// ExtractResponse cuerpo JSON exitoso de POST /v1/extract.
type ExtractResponse struct {
	Artifacts []NormalizedArtifact `json:"artifacts"`
	Warnings  []string             `json:"warnings,omitempty"`
}
