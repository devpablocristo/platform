package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	CapabilityManifestSchemaVersion = "capability_manifest.v1"

	CapabilityTenantScopeGlobal  = "global"
	CapabilityTenantScopeOrg     = "org"
	CapabilityTenantScopeProject = "project"

	CapabilityModeRead  = "read"
	CapabilityModeWrite = "write"

	CapabilityRiskLow      = "low"
	CapabilityRiskMedium   = "medium"
	CapabilityRiskHigh     = "high"
	CapabilityRiskCritical = "critical"
)

var (
	capabilitySlugPattern    = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)
	capabilityToolPattern    = regexp.MustCompile(`^[a-z][a-z0-9_-]*(?:\.[a-z][a-z0-9_-]*)+$`)
	capabilitySemverPattern  = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)
	capabilityTenantScopes   = setOf(CapabilityTenantScopeGlobal, CapabilityTenantScopeOrg, CapabilityTenantScopeProject)
	capabilityModes          = setOf(CapabilityModeRead, CapabilityModeWrite)
	capabilityRiskClasses    = setOf(CapabilityRiskLow, CapabilityRiskMedium, CapabilityRiskHigh, CapabilityRiskCritical)
	forbiddenConfigKeyTokens = []string{
		"apikey",
		"basicauth",
		"baseurl",
		"credential",
		"dsn",
		"endpoint",
		"host",
		"password",
		"secret",
		"token",
		"url",
	}
)

// CapabilityManifest describes a versioned product capability package.
type CapabilityManifest struct {
	SchemaVersion string                      `json:"schema_version"`
	ID            string                      `json:"id"`
	Product       string                      `json:"product"`
	Version       string                      `json:"version"`
	TenantScope   string                      `json:"tenant_scope"`
	Name          string                      `json:"name"`
	Description   string                      `json:"description"`
	Agents        []CapabilityAgentDescriptor `json:"agents"`
	Tools         []CapabilityTool            `json:"tools"`
}

// CapabilityAgentDescriptor is the light routing descriptor exposed by a manifest.
type CapabilityAgentDescriptor struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CapabilityAuthz declares tenant-local role and module requirements.
type CapabilityAuthz struct {
	RequiredRoles   []string `json:"required_roles"`
	RequiredModules []string `json:"required_modules"`
}

// CapabilityExecutor is a logical execution reference. It is not a URL.
type CapabilityExecutor struct {
	ExecutorRef string `json:"executor_ref"`
}

// CapabilityGovernance declares the Nexus Governance metadata for a tool.
type CapabilityGovernance struct {
	RequiresApproval bool   `json:"requires_approval"`
	ActionType     string `json:"action_type,omitempty"`
	TargetSystem   string `json:"target_system,omitempty"`
}

// CapabilityTool describes a tool that a product capability exposes to Companion.
type CapabilityTool struct {
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Mode           string         `json:"mode"`
	SideEffect     bool           `json:"side_effect"`
	RiskClass      string         `json:"risk_class"`
	InputSchema    map[string]any `json:"input_schema"`
	OutputSchema   map[string]any `json:"output_schema,omitempty"`
	EvidenceFields []string       `json:"evidence_fields"`
	CapabilityAuthz
	CapabilityExecutor
	Governance *CapabilityGovernance `json:"governance,omitempty"`
}

// ToTool converts a capability tool into the runtime LLM tool declaration.
func (t CapabilityTool) ToTool() Tool {
	return Tool{
		Name:        t.Name,
		Description: t.Description,
		Parameters:  t.InputSchema,
	}
}

// Tools converts all manifest tools into runtime LLM tool declarations.
func (m CapabilityManifest) ToolsForLLM() []Tool {
	out := make([]Tool, 0, len(m.Tools))
	for _, tool := range m.Tools {
		out = append(out, tool.ToTool())
	}
	return out
}

// ValidateCapabilityManifest validates the hard rules for capability manifests.
func ValidateCapabilityManifest(m CapabilityManifest) error {
	if strings.TrimSpace(m.SchemaVersion) != CapabilityManifestSchemaVersion {
		return fmt.Errorf("schema_version must be %q", CapabilityManifestSchemaVersion)
	}
	if err := validateCapabilitySlug("id", m.ID); err != nil {
		return err
	}
	if err := validateCapabilitySlug("product", m.Product); err != nil {
		return err
	}
	if !capabilitySemverPattern.MatchString(strings.TrimSpace(m.Version)) {
		return fmt.Errorf("version must be semver without leading v")
	}
	if !capabilityTenantScopes[m.TenantScope] {
		return fmt.Errorf("tenant_scope must be one of global, org, project")
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(m.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if len(m.Agents) == 0 {
		return fmt.Errorf("at least one agent descriptor is required")
	}
	if len(m.Tools) == 0 {
		return fmt.Errorf("at least one tool is required")
	}

	agents := map[string]bool{}
	for _, agent := range m.Agents {
		if err := validateCapabilitySlug("agent.name", agent.Name); err != nil {
			return err
		}
		if strings.TrimSpace(agent.Description) == "" {
			return fmt.Errorf("agent %q description is required", agent.Name)
		}
		if agents[agent.Name] {
			return fmt.Errorf("duplicate agent name %q", agent.Name)
		}
		agents[agent.Name] = true
	}

	toolNames := map[string]bool{}
	for _, tool := range m.Tools {
		if toolNames[tool.Name] {
			return fmt.Errorf("duplicate tool name %q", tool.Name)
		}
		toolNames[tool.Name] = true
		if err := validateCapabilityTool(tool); err != nil {
			return err
		}
	}

	payload, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest for config scan: %w", err)
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return fmt.Errorf("decode manifest for config scan: %w", err)
	}
	if err := rejectForbiddenCapabilityConfig(decoded, "manifest"); err != nil {
		return err
	}

	return nil
}

func validateCapabilityTool(t CapabilityTool) error {
	if !capabilityToolPattern.MatchString(strings.TrimSpace(t.Name)) {
		return fmt.Errorf("tool.name %q must use dot notation", t.Name)
	}
	if strings.TrimSpace(t.Description) == "" {
		return fmt.Errorf("tool %q description is required", t.Name)
	}
	if !capabilityModes[t.Mode] {
		return fmt.Errorf("tool %q mode must be read or write", t.Name)
	}
	if !capabilityRiskClasses[t.RiskClass] {
		return fmt.Errorf("tool %q risk_class must be low, medium, high, or critical", t.Name)
	}
	if err := validateJSONSchemaObject("input_schema", t.InputSchema); err != nil {
		return fmt.Errorf("tool %q %w", t.Name, err)
	}
	if len(t.OutputSchema) > 0 {
		if err := validateJSONSchemaObject("output_schema", t.OutputSchema); err != nil {
			return fmt.Errorf("tool %q %w", t.Name, err)
		}
	}
	if strings.TrimSpace(t.ExecutorRef) == "" {
		return fmt.Errorf("tool %q executor_ref is required", t.Name)
	}
	if err := validateNonEmptyList("required_roles", t.RequiredRoles); err != nil {
		return fmt.Errorf("tool %q %w", t.Name, err)
	}
	if err := validateNonEmptyList("required_modules", t.RequiredModules); err != nil {
		return fmt.Errorf("tool %q %w", t.Name, err)
	}
	if err := validateNonEmptyList("evidence_fields", t.EvidenceFields); err != nil {
		return fmt.Errorf("tool %q %w", t.Name, err)
	}

	switch t.Mode {
	case CapabilityModeRead:
		if t.SideEffect {
			return fmt.Errorf("tool %q mode=read requires side_effect=false", t.Name)
		}
		if t.Governance != nil && t.Governance.RequiresApproval {
			return fmt.Errorf("tool %q read tools must not require review", t.Name)
		}
	case CapabilityModeWrite:
		if !t.SideEffect {
			return fmt.Errorf("tool %q mode=write requires side_effect=true", t.Name)
		}
		if len(t.EvidenceFields) == 0 {
			return fmt.Errorf("tool %q write tools require evidence_fields", t.Name)
		}
		if t.Governance == nil {
			return fmt.Errorf("tool %q write tools require governance", t.Name)
		}
		if !t.Governance.RequiresApproval {
			return fmt.Errorf("tool %q write tools require governance.requires_approval=true", t.Name)
		}
		if strings.TrimSpace(t.Governance.ActionType) == "" {
			return fmt.Errorf("tool %q write tools require governance.action_type", t.Name)
		}
	}

	return nil
}

func validateCapabilitySlug(field, value string) error {
	if !capabilitySlugPattern.MatchString(strings.TrimSpace(value)) {
		return fmt.Errorf("%s %q must be a stable slug", field, value)
	}
	return nil
}

func validateJSONSchemaObject(field string, schema map[string]any) error {
	if len(schema) == 0 {
		return fmt.Errorf("%s is required", field)
	}
	if schemaType, ok := schema["type"]; !ok || schemaType != "object" {
		return fmt.Errorf("%s must be a JSON Schema object with type=object", field)
	}
	return nil
}

func validateNonEmptyList(field string, values []string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s cannot contain empty values", field)
		}
	}
	return nil
}

func rejectForbiddenCapabilityConfig(value any, path string) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isForbiddenCapabilityConfigKey(key) {
				return fmt.Errorf("capability manifest contains forbidden configuration key %q at %s", key, path)
			}
			if err := rejectForbiddenCapabilityConfig(child, path+"."+key); err != nil {
				return err
			}
		}
	case []any:
		for i, child := range typed {
			if err := rejectForbiddenCapabilityConfig(child, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case string:
		if looksLikeRuntimeAddress(typed) {
			return fmt.Errorf("capability manifest contains runtime address at %s", path)
		}
	}
	return nil
}

func isForbiddenCapabilityConfigKey(key string) bool {
	normalized := strings.NewReplacer("_", "", "-", "", ".", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	for _, token := range forbiddenConfigKeyTokens {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func looksLikeRuntimeAddress(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(normalized, "://")
}

func setOf(values ...string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}
