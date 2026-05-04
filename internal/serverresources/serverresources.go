package serverresources

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type Registry struct {
	SchemaVersion int        `json:"schema_version"`
	Resources     []Resource `json:"resources"`
	UpdatedAt     string     `json:"updated_at"`
}

type Resource struct {
	ID                string      `json:"id"`
	Environment       string      `json:"environment"`
	Host              string      `json:"host"`
	Provider          string      `json:"provider"`
	Region            string      `json:"region,omitempty"`
	InstanceID        string      `json:"instance_id,omitempty"`
	Owner             string      `json:"owner"`
	Purpose           string      `json:"purpose,omitempty"`
	AuthRef           string      `json:"auth_ref"`
	ExpiresAt         string      `json:"expires_at,omitempty"`
	Spec              ServerSpec  `json:"spec"`
	Healthcheck       Healthcheck `json:"healthcheck"`
	Status            string      `json:"status"`
	ExpirationState   string      `json:"expiration_state"`
	MaintenanceWindow string      `json:"maintenance_window,omitempty"`
	RenewedAt         string      `json:"renewed_at,omitempty"`
	RenewedBy         string      `json:"renewed_by,omitempty"`
	RetiredAt         string      `json:"retired_at,omitempty"`
	RetiredBy         string      `json:"retired_by,omitempty"`
	RetireReason      string      `json:"retire_reason,omitempty"`
	CreatedAt         string      `json:"created_at"`
	UpdatedAt         string      `json:"updated_at"`
}

type ServerSpec struct {
	CPU      int    `json:"cpu,omitempty"`
	MemoryGB int    `json:"memory_gb,omitempty"`
	DiskGB   int    `json:"disk_gb,omitempty"`
	OS       string `json:"os,omitempty"`
}

type Healthcheck struct {
	Type       string `json:"type,omitempty"`
	Target     string `json:"target,omitempty"`
	LastStatus string `json:"last_status,omitempty"`
}

type HealthScanOptions struct {
	Environment string   `json:"environment"`
	ResourceIDs []string `json:"resource_ids"`
	Approved    bool     `json:"approved"`
}

type HealthScanReport struct {
	ID          string             `json:"id"`
	Status      string             `json:"status"`
	Decision    string             `json:"decision"`
	Environment string             `json:"environment,omitempty"`
	Results     []HealthScanResult `json:"results"`
	Reasons     []string           `json:"reasons"`
	CreatedAt   string             `json:"created_at"`
}

type HealthScanResult struct {
	ResourceID  string `json:"resource_id"`
	Environment string `json:"environment"`
	Target      string `json:"target,omitempty"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
	HTTPStatus  int    `json:"http_status,omitempty"`
}

type MaintenanceRecord struct {
	ID              string `json:"id"`
	ResourceID      string `json:"resource_id"`
	Environment     string `json:"environment"`
	Type            string `json:"type"`
	Status          string `json:"status"`
	Decision        string `json:"decision"`
	ExpirationState string `json:"expiration_state,omitempty"`
	ExpiresAt       string `json:"expires_at,omitempty"`
	NewExpiresAt    string `json:"new_expires_at,omitempty"`
	HealthStatus    string `json:"health_status,omitempty"`
	ActorID         string `json:"actor_id,omitempty"`
	Reason          string `json:"reason,omitempty"`
	CreatedAt       string `json:"created_at"`
}

type RenewalOptions struct {
	ResourceID string `json:"resource_id"`
	ExpiresAt  string `json:"expires_at"`
	ActorID    string `json:"actor_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type RetireOptions struct {
	ResourceID string `json:"resource_id"`
	ActorID    string `json:"actor_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

func List(rootDir string) ([]Resource, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return nil, err
	}
	return registry.Resources, nil
}

func Load(rootDir string) (Registry, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Registry{}, err
	}
	var registry Registry
	found, err := fsutil.ReadJSON(inventoryPath(rootDir), &registry)
	if err != nil {
		return Registry{}, err
	}
	if !found || registry.SchemaVersion == 0 {
		registry = Registry{SchemaVersion: 1, Resources: []Resource{}, UpdatedAt: now()}
		if err := save(rootDir, registry); err != nil {
			return Registry{}, err
		}
	}
	registry.Resources = normalizeOrder(registry.Resources)
	return registry, nil
}

func Add(rootDir string, resource Resource) (Resource, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Resource{}, err
	}
	resource, err = normalizeForSave(resource)
	if err != nil {
		return Resource{}, err
	}
	for _, existing := range registry.Resources {
		if existing.ID == resource.ID {
			return Resource{}, errors.New("resource_id_duplicate")
		}
	}
	resource.CreatedAt = now()
	resource.UpdatedAt = resource.CreatedAt
	resource.Status = "active"
	resource.ExpirationState = expirationState(resource.ExpiresAt, time.Now())
	registry.Resources = append(registry.Resources, resource)
	if err := save(rootDir, registry); err != nil {
		return Resource{}, err
	}
	if err := fsutil.AppendJSONL(eventsPath(rootDir), map[string]any{"event": "resource.added", "resource_id": resource.ID, "environment": resource.Environment, "ts": now()}); err != nil {
		return Resource{}, err
	}
	_ = logging.Log(rootDir, "audit", "server_resource.added", map[string]any{"resource_id": resource.ID, "environment": resource.Environment})
	return resource, nil
}

func Show(rootDir string, id string) (Resource, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Resource{}, false, err
	}
	id = normalizeID(id)
	for _, resource := range registry.Resources {
		if resource.ID == id {
			return resource, true, nil
		}
	}
	return Resource{}, false, nil
}

func Disable(rootDir string, id string) (Resource, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Resource{}, false, err
	}
	id = normalizeID(id)
	for idx, resource := range registry.Resources {
		if resource.ID != id {
			continue
		}
		resource.Status = "disabled"
		resource.UpdatedAt = now()
		registry.Resources[idx] = resource
		if err := save(rootDir, registry); err != nil {
			return Resource{}, false, err
		}
		if err := fsutil.AppendJSONL(eventsPath(rootDir), map[string]any{"event": "resource.disabled", "resource_id": resource.ID, "ts": now()}); err != nil {
			return Resource{}, false, err
		}
		_ = logging.Log(rootDir, "audit", "server_resource.disabled", map[string]any{"resource_id": resource.ID})
		return resource, true, nil
	}
	return Resource{}, false, nil
}

func ExpirationScan(rootDir string) ([]Resource, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return nil, err
	}
	nowTime := time.Now()
	result := []Resource{}
	changed := false
	for idx, resource := range registry.Resources {
		state := expirationState(resource.ExpiresAt, nowTime)
		if resource.ExpirationState != state {
			resource.ExpirationState = state
			resource.UpdatedAt = now()
			registry.Resources[idx] = resource
			changed = true
		}
		if state != "unknown" && state != "ok" {
			result = append(result, resource)
		}
	}
	if changed {
		if err := save(rootDir, registry); err != nil {
			return nil, err
		}
	}
	return normalizeOrder(result), nil
}

func HealthScan(ctx context.Context, rootDir string, options HealthScanOptions) (HealthScanReport, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return HealthScanReport{}, err
	}
	options.Environment = normalizeToken(options.Environment)
	options.ResourceIDs = normalizeIDs(options.ResourceIDs)
	report := HealthScanReport{
		ID:          "health-scan-" + timeID(time.Now().UTC()),
		Status:      "completed",
		Decision:    "HEALTH_SCAN_COMPLETED",
		Environment: options.Environment,
		Results:     []HealthScanResult{},
		Reasons:     []string{},
		CreatedAt:   now(),
	}
	changed := false
	for idx, resource := range registry.Resources {
		if len(options.ResourceIDs) > 0 && !contains(options.ResourceIDs, resource.ID) {
			continue
		}
		if options.Environment != "" && resource.Environment != options.Environment {
			continue
		}
		result := scanResource(ctx, resource, options.Approved)
		report.Results = append(report.Results, result)
		if result.Status == "blocked" || result.Status == "failed" {
			report.Status = "attention_required"
			report.Decision = "HEALTH_SCAN_ATTENTION_REQUIRED"
			report.Reasons = append(report.Reasons, result.Reason)
		}
		if result.Status == "healthy" || result.Status == "failed" || result.Status == "blocked" || result.Status == "unknown" {
			registry.Resources[idx].Healthcheck.LastStatus = result.Status
			registry.Resources[idx].UpdatedAt = now()
			changed = true
		}
	}
	if len(report.Results) == 0 {
		report.Status = "blocked"
		report.Decision = "HEALTH_SCAN_BLOCKED"
		report.Reasons = append(report.Reasons, "resource_not_found")
	}
	if changed {
		if err := save(rootDir, registry); err != nil {
			return HealthScanReport{}, err
		}
	}
	if err := fsutil.WriteJSON(healthScanPath(rootDir, report.ID), report); err != nil {
		return HealthScanReport{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).ResourcesDir, "checks.jsonl"), report); err != nil {
		return HealthScanReport{}, err
	}
	_ = logging.Log(rootDir, "audit", "server_resource.health_scan", map[string]any{"scan_id": report.ID, "decision": report.Decision, "status": report.Status, "results": len(report.Results)})
	return report, nil
}

func MaintenanceScan(rootDir string) ([]MaintenanceRecord, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return nil, err
	}
	records := []MaintenanceRecord{}
	changed := false
	nowTime := time.Now()
	for idx, resource := range registry.Resources {
		if resource.Status == "retired" {
			continue
		}
		state := expirationState(resource.ExpiresAt, nowTime)
		if registry.Resources[idx].ExpirationState != state {
			registry.Resources[idx].ExpirationState = state
			registry.Resources[idx].UpdatedAt = now()
			changed = true
		}
		if state == "expired" || state == "critical" || state == "warning" {
			record := newMaintenanceRecord(resource, "expiration_alert", "open", "MAINTENANCE_REQUIRED", "expiration_"+state)
			record.ExpirationState = state
			record.ExpiresAt = resource.ExpiresAt
			records = append(records, record)
		}
		if resource.Healthcheck.LastStatus == "failed" || resource.Healthcheck.LastStatus == "blocked" {
			record := newMaintenanceRecord(resource, "health_attention", "open", "MAINTENANCE_REQUIRED", "health_"+resource.Healthcheck.LastStatus)
			record.HealthStatus = resource.Healthcheck.LastStatus
			records = append(records, record)
		}
	}
	if changed {
		if err := save(rootDir, registry); err != nil {
			return nil, err
		}
	}
	for _, record := range records {
		if err := saveMaintenanceRecord(rootDir, record); err != nil {
			return nil, err
		}
	}
	if len(records) > 0 {
		_ = logging.Log(rootDir, "audit", "server_resource.maintenance_scan", map[string]any{"records": len(records), "decision": "MAINTENANCE_REQUIRED"})
	}
	sortMaintenance(records)
	return records, nil
}

func ListMaintenance(rootDir string, limit int) ([]MaintenanceRecord, error) {
	if err := fsutil.EnsureDir(maintenanceDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(maintenanceDir(rootDir))
	if err != nil {
		return nil, err
	}
	records := []MaintenanceRecord{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var record MaintenanceRecord
		found, err := fsutil.ReadJSON(filepath.Join(maintenanceDir(rootDir), entry.Name()), &record)
		if err != nil {
			return nil, err
		}
		if found && record.ID != "" {
			records = append(records, record)
		}
	}
	sortMaintenance(records)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if len(records) > limit {
		return records[:limit], nil
	}
	return records, nil
}

func Renew(rootDir string, options RenewalOptions) (Resource, MaintenanceRecord, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Resource{}, MaintenanceRecord{}, false, err
	}
	options.ResourceID = normalizeID(options.ResourceID)
	options.ActorID = normalizeActor(options.ActorID)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.ResourceID == "" {
		return Resource{}, MaintenanceRecord{}, false, errors.New("resource_id_required")
	}
	if _, err := time.Parse("2006-01-02", options.ExpiresAt); err != nil {
		return Resource{}, MaintenanceRecord{}, false, errors.New("expires_at_must_be_date")
	}
	for idx, resource := range registry.Resources {
		if resource.ID != options.ResourceID {
			continue
		}
		resource.ExpiresAt = options.ExpiresAt
		resource.ExpirationState = expirationState(resource.ExpiresAt, time.Now())
		resource.RenewedAt = now()
		resource.RenewedBy = options.ActorID
		resource.UpdatedAt = resource.RenewedAt
		registry.Resources[idx] = resource
		if err := save(rootDir, registry); err != nil {
			return Resource{}, MaintenanceRecord{}, true, err
		}
		record := newMaintenanceRecord(resource, "renewal_recorded", "completed", "RESOURCE_RENEWAL_RECORDED", options.Reason)
		record.NewExpiresAt = options.ExpiresAt
		record.ActorID = options.ActorID
		if err := saveMaintenanceRecord(rootDir, record); err != nil {
			return Resource{}, MaintenanceRecord{}, true, err
		}
		_ = logging.Log(rootDir, "audit", "server_resource.renewed", map[string]any{"resource_id": resource.ID, "expires_at": resource.ExpiresAt, "actor_id": options.ActorID})
		return resource, record, true, nil
	}
	return Resource{}, MaintenanceRecord{}, false, nil
}

func Retire(rootDir string, options RetireOptions) (Resource, MaintenanceRecord, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Resource{}, MaintenanceRecord{}, false, err
	}
	options.ResourceID = normalizeID(options.ResourceID)
	options.ActorID = normalizeActor(options.ActorID)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.ResourceID == "" {
		return Resource{}, MaintenanceRecord{}, false, errors.New("resource_id_required")
	}
	for idx, resource := range registry.Resources {
		if resource.ID != options.ResourceID {
			continue
		}
		resource.Status = "retired"
		resource.RetiredAt = now()
		resource.RetiredBy = options.ActorID
		resource.RetireReason = options.Reason
		resource.UpdatedAt = resource.RetiredAt
		registry.Resources[idx] = resource
		if err := save(rootDir, registry); err != nil {
			return Resource{}, MaintenanceRecord{}, true, err
		}
		record := newMaintenanceRecord(resource, "retirement_recorded", "completed", "RESOURCE_RETIRED", options.Reason)
		record.ActorID = options.ActorID
		if err := saveMaintenanceRecord(rootDir, record); err != nil {
			return Resource{}, MaintenanceRecord{}, true, err
		}
		_ = logging.Log(rootDir, "audit", "server_resource.retired", map[string]any{"resource_id": resource.ID, "actor_id": options.ActorID})
		return resource, record, true, nil
	}
	return Resource{}, MaintenanceRecord{}, false, nil
}

func save(rootDir string, registry Registry) error {
	registry.SchemaVersion = 1
	registry.Resources = normalizeOrder(registry.Resources)
	registry.UpdatedAt = now()
	return fsutil.WriteJSON(inventoryPath(rootDir), registry)
}

func normalizeForSave(resource Resource) (Resource, error) {
	resource.ID = normalizeID(resource.ID)
	resource.Environment = normalizeToken(resource.Environment)
	resource.Provider = normalizeToken(resource.Provider)
	resource.AuthRef = strings.TrimSpace(resource.AuthRef)
	resource.Host = strings.TrimSpace(resource.Host)
	resource.Owner = strings.TrimSpace(resource.Owner)
	resource.Region = strings.TrimSpace(resource.Region)
	resource.InstanceID = strings.TrimSpace(resource.InstanceID)
	resource.Purpose = strings.TrimSpace(resource.Purpose)
	resource.MaintenanceWindow = strings.TrimSpace(resource.MaintenanceWindow)
	resource.Healthcheck.Type = normalizeToken(resource.Healthcheck.Type)
	resource.Healthcheck.Target = strings.TrimSpace(resource.Healthcheck.Target)
	resource.Healthcheck.LastStatus = normalizeToken(resource.Healthcheck.LastStatus)
	if resource.ID == "" {
		return Resource{}, errors.New("resource_id_required")
	}
	if resource.Environment == "" {
		return Resource{}, errors.New("environment_required")
	}
	if !allowedEnvironment(resource.Environment) {
		return Resource{}, errors.New("environment_invalid")
	}
	if resource.Host == "" {
		return Resource{}, errors.New("host_required")
	}
	if resource.Provider == "" {
		return Resource{}, errors.New("provider_required")
	}
	if resource.Owner == "" {
		return Resource{}, errors.New("owner_required")
	}
	if resource.AuthRef == "" {
		return Resource{}, errors.New("auth_ref_required")
	}
	if !isSafeRef(resource.AuthRef) {
		return Resource{}, errors.New("auth_ref_must_be_reference")
	}
	if resource.Environment == "production" && resource.ExpiresAt == "" {
		return Resource{}, errors.New("production_expires_at_required")
	}
	if resource.ExpiresAt != "" {
		if _, err := time.Parse("2006-01-02", resource.ExpiresAt); err != nil {
			return Resource{}, errors.New("expires_at_must_be_date")
		}
	}
	if resource.Healthcheck.Type == "" {
		resource.Healthcheck.Type = "manual"
	}
	if resource.Healthcheck.LastStatus == "" {
		resource.Healthcheck.LastStatus = "unknown"
	}
	return resource, nil
}

func inventoryPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ResourcesDir, "inventory.json")
}

func eventsPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ResourcesDir, "events.jsonl")
}

func healthScanPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ResourcesDir, "checks", id+".json")
}

func maintenanceDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ResourcesDir, "maintenance")
}

func maintenanceRecordPath(rootDir string, id string) string {
	return filepath.Join(maintenanceDir(rootDir), id+".json")
}

func saveMaintenanceRecord(rootDir string, record MaintenanceRecord) error {
	if err := fsutil.WriteJSON(maintenanceRecordPath(rootDir, record.ID), record); err != nil {
		return err
	}
	return fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).ResourcesDir, "maintenance.jsonl"), record)
}

func newMaintenanceRecord(resource Resource, recordType string, status string, decision string, reason string) MaintenanceRecord {
	createdAt := now()
	return MaintenanceRecord{
		ID:          "maintenance-" + normalizeID(resource.ID) + "-" + normalizeToken(recordType) + "-" + timeID(time.Now().UTC()),
		ResourceID:  resource.ID,
		Environment: resource.Environment,
		Type:        normalizeToken(recordType),
		Status:      normalizeToken(status),
		Decision:    decision,
		Reason:      strings.TrimSpace(reason),
		CreatedAt:   createdAt,
	}
}

func scanResource(ctx context.Context, resource Resource, approved bool) HealthScanResult {
	result := HealthScanResult{ResourceID: resource.ID, Environment: resource.Environment, Target: resource.Healthcheck.Target, Status: "unknown"}
	if resource.Environment == "production" && !approved {
		result.Status = "blocked"
		result.Reason = "production_approval_required"
		return result
	}
	if resource.Status != "active" {
		result.Status = "blocked"
		result.Reason = "resource_not_active"
		return result
	}
	switch resource.Healthcheck.Type {
	case "manual":
		result.Status = "unknown"
		result.Reason = "manual_healthcheck"
	case "http", "https":
		return scanHTTP(ctx, resource, result)
	default:
		result.Status = "blocked"
		result.Reason = "healthcheck_type_not_allowed:" + resource.Healthcheck.Type
	}
	return result
}

func scanHTTP(ctx context.Context, resource Resource, result HealthScanResult) HealthScanResult {
	target := strings.TrimSpace(resource.Healthcheck.Target)
	if target == "" {
		result.Status = "blocked"
		result.Reason = "healthcheck_target_required"
		return result
	}
	parsed, err := url.Parse(target)
	if err != nil || parsed.Hostname() == "" {
		result.Status = "blocked"
		result.Reason = "healthcheck_target_invalid"
		return result
	}
	if parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost" {
		result.Status = "blocked"
		result.Reason = "healthcheck_target_not_allowed"
		return result
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		result.Status = "blocked"
		result.Reason = "healthcheck_request_invalid"
		return result
	}
	client := http.Client{Timeout: 3 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		result.Status = "failed"
		result.Reason = "healthcheck_request_failed"
		return result
	}
	defer response.Body.Close()
	result.HTTPStatus = response.StatusCode
	if response.StatusCode >= 200 && response.StatusCode < 400 {
		result.Status = "healthy"
		return result
	}
	result.Status = "failed"
	result.Reason = "healthcheck_http_status"
	return result
}

func normalizeOrder(resources []Resource) []Resource {
	out := append([]Resource{}, resources...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func normalizeID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	return strings.Trim(value, "-")
}

func normalizeIDs(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = normalizeID(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func normalizeToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.ReplaceAll(value, "-", "_")
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func normalizeActor(value string) string {
	value = normalizeID(value)
	if value == "" {
		return "system"
	}
	return value
}

func sortMaintenance(records []MaintenanceRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].CreatedAt > records[j].CreatedAt
	})
}

func allowedEnvironment(value string) bool {
	return value == "test_dev" || value == "staging" || value == "production"
}

func isSafeRef(value string) bool {
	return (strings.HasPrefix(value, "env:") && len(value) > len("env:")) || (strings.HasPrefix(value, "secret:") && len(value) > len("secret:"))
}

func expirationState(value string, nowTime time.Time) string {
	if value == "" {
		return "unknown"
	}
	expiresAt, err := time.Parse("2006-01-02", value)
	if err != nil {
		return "unknown"
	}
	days := int(expiresAt.Sub(nowTime).Hours() / 24)
	switch {
	case days < 0:
		return "expired"
	case days <= 7:
		return "critical"
	case days <= 30:
		return "warning"
	default:
		return "ok"
	}
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func timeID(value time.Time) string {
	return value.Format("20060102150405") + "-" + value.Format("000000000")
}
