package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"

	"gopkg.in/yaml.v3"
)

type ResolveOptions struct {
	Purpose   string `json:"purpose,omitempty"`
	AdapterID string `json:"adapter_id,omitempty"`
	Required  bool   `json:"required,omitempty"`
}

type Resolution struct {
	Reference string `json:"ref"`
	Source    string `json:"source"`
	Name      string `json:"name"`
	SecretID  string `json:"secret_id,omitempty"`
	Type      string `json:"type,omitempty"`
	Purpose   string `json:"purpose,omitempty"`
	AdapterID string `json:"adapter_id,omitempty"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
	EnvKey    string `json:"env_key,omitempty"`

	value string
}

type policyFile struct {
	SchemaVersion int                    `yaml:"schema_version"`
	Secrets       map[string]policyEntry `yaml:"secrets"`
}

type policyEntry struct {
	Type  string   `yaml:"type"`
	Ref   string   `yaml:"ref"`
	Usage []string `yaml:"usage"`
}

var (
	envNamePattern         = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	secretIDPattern        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	credentialPattern      = regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret|credential)\s*[:=]\s*[^,\s]+`)
	openAIKeyPattern       = regexp.MustCompile(`sk-[A-Za-z0-9_-]{8,}`)
	privateKeyBlockPattern = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`)
)

func Resolve(rootDir string, ref string, options ResolveOptions) (Resolution, error) {
	return resolve(rootDir, ref, options, true)
}

func Status(rootDir string, ref string, options ResolveOptions) (Resolution, error) {
	return resolve(rootDir, ref, options, false)
}

func IsSafeReference(ref string) bool {
	source, name, ok := ParseReference(ref)
	if !ok {
		return false
	}
	switch source {
	case "env":
		return envNamePattern.MatchString(name)
	case "secret":
		return secretIDPattern.MatchString(name)
	default:
		return false
	}
}

func ParseReference(ref string) (string, string, bool) {
	ref = strings.TrimSpace(ref)
	source, name, ok := strings.Cut(ref, ":")
	if !ok {
		return "", "", false
	}
	source = strings.TrimSpace(strings.ToLower(source))
	name = strings.TrimSpace(name)
	if source == "" || name == "" {
		return source, name, false
	}
	return source, name, true
}

func Redact(value string) string {
	value = privateKeyBlockPattern.ReplaceAllString(value, "-----BEGIN PRIVATE KEY-----[REDACTED]-----END PRIVATE KEY-----")
	value = credentialPattern.ReplaceAllString(value, "$1=[REDACTED]")
	value = openAIKeyPattern.ReplaceAllString(value, "sk-[REDACTED]")
	return value
}

func (r Resolution) Value() string {
	return r.value
}

func (r Resolution) String() string {
	return fmt.Sprintf("secret_resolution{ref:%q source:%q secret_id:%q status:%q reason:%q env_key:%q purpose:%q adapter_id:%q}", r.Reference, r.Source, r.SecretID, r.Status, r.Reason, r.EnvKey, r.Purpose, r.AdapterID)
}

func (r Resolution) GoString() string {
	return r.String()
}

func resolve(rootDir string, ref string, options ResolveOptions, includeValue bool) (Resolution, error) {
	options.Purpose = normalizePurpose(options.Purpose)
	options.AdapterID = strings.TrimSpace(options.AdapterID)
	ref = strings.TrimSpace(ref)
	source, name, ok := ParseReference(ref)
	resolution := Resolution{Reference: ref, Source: source, Name: name, Purpose: options.Purpose, AdapterID: options.AdapterID}
	if !ok {
		resolution.Status = "invalid"
		if ref == "" {
			resolution.Status = "missing"
			resolution.Reason = "secret_ref_missing"
		} else {
			resolution.Reason = "secret_ref_invalid"
		}
		audit(rootDir, includeValue, resolution)
		return resolution, nil
	}
	switch source {
	case "env":
		resolution = resolveEnv(resolution, name, includeValue)
	case "secret":
		var err error
		resolution, err = resolveSecret(rootDir, resolution, name, options, includeValue)
		if err != nil {
			return Resolution{}, err
		}
	default:
		resolution.Status = "invalid"
		resolution.Reason = "secret_ref_unsupported:" + source
	}
	audit(rootDir, includeValue, resolution)
	return resolution, nil
}

func resolveEnv(base Resolution, key string, includeValue bool) Resolution {
	base.EnvKey = key
	if !envNamePattern.MatchString(key) {
		base.Status = "invalid"
		base.Reason = "secret_env_key_invalid:" + key
		return base
	}
	value := os.Getenv(key)
	if value == "" {
		base.Status = "missing"
		base.Reason = "secret_env_missing:" + key
		return base
	}
	base.Status = "ok"
	base.Reason = "secret_env_present:" + key
	if includeValue {
		base.value = value
	}
	return base
}

func resolveSecret(rootDir string, base Resolution, id string, options ResolveOptions, includeValue bool) (Resolution, error) {
	base.SecretID = id
	if !secretIDPattern.MatchString(id) {
		base.Status = "invalid"
		base.Reason = "secret_id_invalid:" + id
		return base, nil
	}
	policy, found, err := loadPolicy(rootDir)
	if err != nil {
		return Resolution{}, err
	}
	if !found {
		base.Status = "missing"
		base.Reason = "secret_policy_missing"
		return base, nil
	}
	entry, ok := policy.Secrets[id]
	if !ok {
		base.Status = "missing"
		base.Reason = "secret_not_registered:" + id
		return base, nil
	}
	base.Type = strings.TrimSpace(entry.Type)
	if len(entry.Usage) == 0 {
		base.Status = "invalid"
		base.Reason = "secret_usage_required:" + id
		return base, nil
	}
	if options.Purpose == "" {
		base.Status = "denied"
		base.Reason = "secret_usage_purpose_required:" + id
		return base, nil
	}
	if !usageAllowed(entry.Usage, options.Purpose) {
		base.Status = "denied"
		base.Reason = "secret_usage_not_allowed:" + id + ":" + options.Purpose
		return base, nil
	}
	entryRef := strings.TrimSpace(entry.Ref)
	source, name, ok := ParseReference(entryRef)
	if !ok {
		base.Status = "invalid"
		base.Reason = "secret_entry_ref_invalid:" + id
		return base, nil
	}
	if source == "secret" {
		base.Status = "invalid"
		base.Reason = "secret_entry_nested_ref_not_allowed:" + id
		return base, nil
	}
	if source != "env" {
		base.Status = "invalid"
		base.Reason = "secret_backend_unsupported:" + source
		return base, nil
	}
	resolved := resolveEnv(base, name, includeValue)
	if resolved.Status == "ok" {
		resolved.Reason = "secret_resolved_from_env:" + id + ":" + name
	} else if resolved.Reason != "" {
		resolved.Reason = strings.Replace(resolved.Reason, "secret_env_", "secret_entry_env_", 1)
	}
	return resolved, nil
}

func loadPolicy(rootDir string) (policyFile, bool, error) {
	path := filepath.Join(workspace.ForRoot(rootDir).MoyuanDir, "policies", "secrets.yaml")
	text, found, err := fsutil.ReadText(path)
	if err != nil || !found {
		return policyFile{}, found, err
	}
	var policy policyFile
	if err := yaml.Unmarshal([]byte(text), &policy); err != nil {
		return policyFile{}, true, err
	}
	if policy.Secrets == nil {
		policy.Secrets = map[string]policyEntry{}
	}
	return policy, true, nil
}

func usageAllowed(allowed []string, purpose string) bool {
	purpose = normalizePurpose(purpose)
	for _, item := range allowed {
		item = normalizePurpose(item)
		if item == "" {
			continue
		}
		if item == "*" || item == purpose {
			return true
		}
		if strings.HasSuffix(item, ".*") && strings.HasPrefix(purpose, strings.TrimSuffix(item, "*")) {
			return true
		}
	}
	return false
}

func audit(rootDir string, enabled bool, resolution Resolution) {
	if !enabled {
		return
	}
	event := "secret.access.denied"
	if resolution.Status == "ok" {
		event = "secret.access.granted"
	}
	_ = logging.Log(rootDir, "audit", event, map[string]any{
		"ref":        resolution.Reference,
		"source":     resolution.Source,
		"secret_id":  resolution.SecretID,
		"type":       resolution.Type,
		"purpose":    resolution.Purpose,
		"adapter_id": resolution.AdapterID,
		"status":     resolution.Status,
		"reason":     resolution.Reason,
		"env_key":    resolution.EnvKey,
	})
}

func normalizePurpose(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func NormalizeUsages(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = normalizePurpose(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
