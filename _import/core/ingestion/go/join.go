package ingestion

import "strings"

// JoinFullText concatena los FullText de los artifacts con doble salto de línea entre fragmentos no vacíos.
func JoinFullText(arts []NormalizedArtifact) string {
	var b strings.Builder
	for _, a := range arts {
		if a.FullText == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(a.FullText)
	}
	return b.String()
}

// JoinProvenance resume motores en forma `engine` o `engine:version`, separados por `;`.
func JoinProvenance(arts []NormalizedArtifact) string {
	var parts []string
	for _, a := range arts {
		if a.Provenance.Engine == "" {
			continue
		}
		if a.Provenance.EngineVersion != "" {
			parts = append(parts, a.Provenance.Engine+":"+a.Provenance.EngineVersion)
		} else {
			parts = append(parts, a.Provenance.Engine)
		}
	}
	return strings.Join(parts, ";")
}
