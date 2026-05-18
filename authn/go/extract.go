package authn

// DefaultHeaderExtractor implementa Extractor con la prioridad habitual:
// Authorization Bearer primero; si no hay Bearer, X-API-Key o Authorization ApiKey.
//
// No modela el fallback “JWT inválido → probar API key”; eso está en TryInbound.
type DefaultHeaderExtractor struct{}

// Extract implementa Extractor.
func (DefaultHeaderExtractor) Extract(authHeader, apiKeyHeader string) (Credential, error) {
	if token, ok := BearerToken(authHeader); ok {
		return BearerCredential{Token: token}, nil
	}
	if key, ok := APIKeyToken(authHeader, apiKeyHeader); ok {
		return APIKeyCredential{Key: key}, nil
	}
	return nil, nil
}
