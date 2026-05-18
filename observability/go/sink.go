package observability

// Sink define una interfaz mínima y reusable para métricas.
type Sink interface {
	IncCounter(name string, labels map[string]string)
	ObserveHistogram(name string, value float64, labels map[string]string)
	SetGauge(name string, value float64, labels map[string]string)
}
