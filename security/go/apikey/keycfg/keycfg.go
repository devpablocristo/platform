// Package keycfg parsea configuraciones de API keys con metadata sidecar.
//
// Formato soportado (compatible con axis/companion + axis/nexus):
//
//	name1=secret1|attr=value|attr=value,name2=secret2|...
//
// El delimitador principal entre entries es `,` o `\n`. Dentro de un entry,
// `|` separa el secreto del metadata. Cada par `attr=value` es case-insensitive
// en la izquierda. Attributes soportados:
//
//   - actor, actor_id, user, user_id   → Metadata.Actor
//   - role                              → Metadata.Role
//   - org, org_id, tenant, tenant_id    → Metadata.OrgID
//   - scope, scopes                     → Metadata.Scopes (separados por space/`+`/`;`)
//   - service, service_principal        → Metadata.ServicePrincipal (bool truthy)
//
// El primer retorno de Parse es el string sanitizado (formato `name=secret,name=secret,...`)
// listo para pasar a `apikey.NewAuthenticator`. El segundo es el map de metadata
// por name. Diseño: separar metadata del secret permite que el authenticator
// base de platform siga siendo agnóstico mientras el caller se queda con el
// contexto rico.
//
// Profiles permiten que cada producto registre listas de scopes por defecto
// para nombres conocidos (e.g. companion registra "admin" → [companion:tasks:*]).
// `DefaultScopesFor` consulta el registry global; thread-safe vía RWMutex.
package keycfg

import (
	"strings"
	"sync"
)

// Metadata describe los atributos sidecar asociados a un API key.
type Metadata struct {
	Actor            string
	Role             string
	OrgID            string
	Scopes           []string
	ServicePrincipal bool
}

// Parse divide un raw config en (sanitized, metadata).
// Sanitized es el formato `name=secret,name=secret,...` para apikey.NewAuthenticator.
// Si un entry no tiene `name=...`, se preserva tal cual en sanitized y no entra
// al map de metadata (downstream apikey.NewAuthenticator lo rechazará si está
// mal formado).
//
// Si `Metadata.Actor` o `Metadata.Role` no fueron seteados explícitamente,
// se defaultean al name (matchea el comportamiento de companion/nexus pre-lift).
func Parse(raw string) (string, map[string]Metadata) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n'
	})
	sanitized := make([]string, 0, len(parts))
	metadata := make(map[string]Metadata, len(parts))
	for _, part := range parts {
		piece := strings.TrimSpace(part)
		if piece == "" {
			continue
		}
		name, rhs, ok := strings.Cut(piece, "=")
		if !ok {
			sanitized = append(sanitized, piece)
			continue
		}
		name = strings.TrimSpace(name)
		rhs = strings.TrimSpace(rhs)
		if name == "" || rhs == "" {
			sanitized = append(sanitized, piece)
			continue
		}
		secret, meta := parseValue(rhs)
		if secret == "" {
			secret = rhs
		}
		sanitized = append(sanitized, name+"="+secret)
		if meta.Actor == "" {
			meta.Actor = name
		}
		if meta.Role == "" {
			meta.Role = name
		}
		metadata[name] = meta
	}
	return strings.Join(sanitized, ","), metadata
}

// parseValue divide `secret|attr=value|attr=value` en (secret, Metadata).
func parseValue(value string) (string, Metadata) {
	segments := strings.Split(value, "|")
	if len(segments) == 0 {
		return "", Metadata{}
	}
	secret := strings.TrimSpace(segments[0])
	meta := Metadata{}
	for _, segment := range segments[1:] {
		key, raw, ok := strings.Cut(strings.TrimSpace(segment), "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(key)) {
		case "actor", "actor_id", "user", "user_id":
			meta.Actor = strings.TrimSpace(raw)
		case "role":
			meta.Role = strings.TrimSpace(raw)
		case "org", "org_id", "tenant", "tenant_id":
			meta.OrgID = strings.TrimSpace(raw)
		case "scope", "scopes":
			meta.Scopes = ParseScopeList(raw)
		case "service", "service_principal":
			meta.ServicePrincipal = parseBool(raw)
		}
	}
	return secret, meta
}

// ParseScopeList convierte un raw string de scopes (separados por space, `+`
// o `;`) en una lista normalizada sin duplicados.
func ParseScopeList(raw string) []string {
	raw = strings.NewReplacer(";", " ", "+", " ").Replace(raw)
	fields := strings.Fields(raw)
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		scope := strings.TrimSpace(field)
		if scope == "" {
			continue
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out
}

func parseBool(raw string) bool {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "1", "true", "yes", "y", "service":
		return true
	default:
		return false
	}
}

// --- Profile registry ---
//
// Cada producto (companion, nexus, pymes, etc) registra al wire-up sus perfiles
// con scopes por defecto. Esto permite que `DefaultScopesFor("admin")` retorne
// la lista correcta sin hardcodear scopes producto-específicos en platform.

// Profile asocia un name (típicamente "admin", "service", etc) con una lista
// de scopes por defecto. La aplicación es producto-específica.
type Profile struct {
	Name   string
	Scopes []string
}

var (
	profileMu  sync.RWMutex
	profileReg = make(map[string][]string)
)

// Register asocia un name con una lista de scopes por defecto. Llamadas
// posteriores con el mismo name sobreescriben. Thread-safe.
func Register(p Profile) {
	name := strings.TrimSpace(strings.ToLower(p.Name))
	if name == "" {
		return
	}
	profileMu.Lock()
	defer profileMu.Unlock()
	dup := make([]string, len(p.Scopes))
	copy(dup, p.Scopes)
	profileReg[name] = dup
}

// DefaultScopesFor retorna los scopes registrados para un name (case-insensitive).
// Si no hay perfil registrado, retorna nil. El caller decide qué hacer con eso
// (típicamente combinar con `Metadata.Scopes` explícitos).
func DefaultScopesFor(name string) []string {
	key := strings.TrimSpace(strings.ToLower(name))
	if key == "" {
		return nil
	}
	profileMu.RLock()
	defer profileMu.RUnlock()
	src, ok := profileReg[key]
	if !ok {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}

// ResetProfiles borra el registry. Pensado para tests; no es seguro llamarlo
// concurrentemente con Register/DefaultScopesFor.
func ResetProfiles() {
	profileMu.Lock()
	defer profileMu.Unlock()
	profileReg = make(map[string][]string)
}
