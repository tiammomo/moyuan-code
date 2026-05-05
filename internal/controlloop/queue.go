package controlloop

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/operations"
	"moyuan-code/internal/workspace"
)

type QueueOptions struct {
	Trigger               string   `json:"trigger,omitempty"`
	RequestedBy           string   `json:"requested_by,omitempty"`
	IdempotencyKey        string   `json:"idempotency_key,omitempty"`
	RetryBudget           int      `json:"retry_budget,omitempty"`
	Steps                 []string `json:"steps,omitempty"`
	Environment           string   `json:"environment,omitempty"`
	ResourceIDs           []string `json:"resource_ids,omitempty"`
	DeploymentExecutionID string   `json:"deployment_execution_id,omitempty"`
	MaintenanceWindow     string   `json:"maintenance_window,omitempty"`
	DueAt                 string   `json:"due_at,omitempty"`
	Priority              int      `json:"priority,omitempty"`
	AdmissionID           string   `json:"admission_id,omitempty"`
	RemoteRehearsalID     string   `json:"remote_rehearsal_id,omitempty"`
	ReviewPacketID        string   `json:"review_packet_id,omitempty"`
}

type QueueListOptions struct {
	Status      string `json:"status,omitempty"`
	Environment string `json:"environment,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type QueueRunOptions struct {
	Status      string `json:"status,omitempty"`
	Environment string `json:"environment,omitempty"`
	MaxItems    int    `json:"max_items,omitempty"`
}

type QueueRunReport struct {
	ID          string      `json:"id"`
	StartedAt   string      `json:"started_at"`
	FinishedAt  string      `json:"finished_at"`
	Status      string      `json:"status"`
	Decision    string      `json:"decision"`
	Processed   int         `json:"processed"`
	Executed    int         `json:"executed"`
	Waiting     int         `json:"waiting"`
	Manual      int         `json:"manual"`
	QueueItems  []QueueItem `json:"queue_items"`
	ControlRuns []RunRecord `json:"control_runs,omitempty"`
}

type QueueItem struct {
	ID                    string   `json:"id"`
	Status                string   `json:"status"`
	Decision              string   `json:"decision"`
	Trigger               string   `json:"trigger"`
	RequestedBy           string   `json:"requested_by,omitempty"`
	IdempotencyKey        string   `json:"idempotency_key,omitempty"`
	RetryBudget           int      `json:"retry_budget"`
	AttemptCount          int      `json:"attempt_count"`
	Steps                 []string `json:"steps"`
	Environment           string   `json:"environment,omitempty"`
	ResourceIDs           []string `json:"resource_ids,omitempty"`
	DeploymentExecutionID string   `json:"deployment_execution_id,omitempty"`
	MaintenanceWindow     string   `json:"maintenance_window,omitempty"`
	DueAt                 string   `json:"due_at,omitempty"`
	Priority              int      `json:"priority,omitempty"`
	AdmissionID           string   `json:"admission_id,omitempty"`
	RemoteRehearsalID     string   `json:"remote_rehearsal_id,omitempty"`
	ReviewPacketID        string   `json:"review_packet_id,omitempty"`
	RunID                 string   `json:"run_id,omitempty"`
	Reasons               []string `json:"reasons,omitempty"`
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
}

func Enqueue(rootDir string, options QueueOptions) (QueueItem, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return QueueItem{}, err
	}
	options = normalizeQueueOptions(options)
	now := time.Now().UTC()
	item := QueueItem{
		ID:                    "control-queue-" + timeID(now),
		Status:                "queued",
		Decision:              "CONTROL_QUEUE_QUEUED",
		Trigger:               options.Trigger,
		RequestedBy:           options.RequestedBy,
		IdempotencyKey:        options.IdempotencyKey,
		RetryBudget:           options.RetryBudget,
		AttemptCount:          0,
		Steps:                 append([]string{}, options.Steps...),
		Environment:           options.Environment,
		ResourceIDs:           append([]string{}, options.ResourceIDs...),
		DeploymentExecutionID: options.DeploymentExecutionID,
		MaintenanceWindow:     options.MaintenanceWindow,
		DueAt:                 options.DueAt,
		Priority:              options.Priority,
		AdmissionID:           options.AdmissionID,
		RemoteRehearsalID:     options.RemoteRehearsalID,
		ReviewPacketID:        options.ReviewPacketID,
		Reasons:               []string{"control_queue_item_created"},
		CreatedAt:             now.Format(time.RFC3339Nano),
		UpdatedAt:             now.Format(time.RFC3339Nano),
	}
	if err := saveQueueItem(rootDir, item); err != nil {
		return QueueItem{}, err
	}
	_ = logging.Log(rootDir, "audit", "control_loop.queue.created", map[string]any{
		"queue_id":    item.ID,
		"environment": item.Environment,
		"steps":       len(item.Steps),
		"window":      item.MaintenanceWindow,
	})
	return item, nil
}

func ListQueue(rootDir string, options QueueListOptions) ([]QueueItem, error) {
	options = normalizeQueueListOptions(options)
	if err := fsutil.EnsureDir(queueDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(queueDir(rootDir))
	if err != nil {
		return nil, err
	}
	items := []QueueItem{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var item QueueItem
		found, err := fsutil.ReadJSON(filepath.Join(queueDir(rootDir), entry.Name()), &item)
		if err != nil {
			return nil, err
		}
		if found && item.ID != "" && queueItemMatches(item, options) {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority > items[j].Priority
		}
		return items[i].CreatedAt < items[j].CreatedAt
	})
	if len(items) > options.Limit {
		items = items[:options.Limit]
	}
	return items, nil
}

func RunQueue(ctx context.Context, rootDir string, options QueueRunOptions) (QueueRunReport, error) {
	options = normalizeQueueRunOptions(options)
	start := time.Now().UTC()
	report := QueueRunReport{
		ID:          "control-queue-run-" + timeID(start),
		StartedAt:   start.Format(time.RFC3339Nano),
		Status:      "running",
		Decision:    "CONTROL_QUEUE_RUN_RUNNING",
		QueueItems:  []QueueItem{},
		ControlRuns: []RunRecord{},
	}
	items, err := ListQueue(rootDir, QueueListOptions{Status: options.Status, Environment: options.Environment, Limit: options.MaxItems})
	if err != nil {
		return QueueRunReport{}, err
	}
	for _, item := range items {
		if item.Status != "queued" && item.Status != "waiting" {
			continue
		}
		item, run, err := executeQueueItem(ctx, rootDir, item)
		if err != nil {
			return QueueRunReport{}, err
		}
		report.Processed++
		report.QueueItems = append(report.QueueItems, item)
		if run.ID != "" {
			report.ControlRuns = append(report.ControlRuns, run)
		}
		switch item.Status {
		case "completed":
			report.Executed++
		case "waiting":
			report.Waiting++
		case "manual_required":
			report.Manual++
		}
	}
	finished := time.Now().UTC()
	report.FinishedAt = finished.Format(time.RFC3339Nano)
	report.Status = "completed"
	report.Decision = "CONTROL_QUEUE_RUN_COMPLETED"
	if report.Waiting > 0 || report.Manual > 0 {
		report.Decision = "CONTROL_QUEUE_RUN_COMPLETED_WITH_ATTENTION"
	}
	if report.Processed == 0 {
		report.Decision = "CONTROL_QUEUE_RUN_NOOP"
	}
	_ = logging.Log(rootDir, "audit", "control_loop.queue.run.completed", map[string]any{
		"queue_run_id": report.ID,
		"processed":    report.Processed,
		"executed":     report.Executed,
		"waiting":      report.Waiting,
		"manual":       report.Manual,
	})
	return report, nil
}

func executeQueueItem(ctx context.Context, rootDir string, item QueueItem) (QueueItem, RunRecord, error) {
	now := time.Now().UTC()
	inWindow, reason := maintenanceWindowAllows(item.MaintenanceWindow, item.DueAt, now)
	if reason == "maintenance_window_invalid" {
		item.Status = "manual_required"
		item.Decision = "CONTROL_QUEUE_MANUAL_HANDOFF"
		item.Reasons = appendUniqueQueueReason(item.Reasons, reason)
		item.UpdatedAt = now.Format(time.RFC3339Nano)
		return item, RunRecord{}, saveQueueItem(rootDir, item)
	}
	if !inWindow {
		item.Status = "waiting"
		item.Decision = "CONTROL_QUEUE_WAITING_MAINTENANCE_WINDOW"
		item.Reasons = appendUniqueQueueReason(item.Reasons, reason)
		item.UpdatedAt = now.Format(time.RFC3339Nano)
		return item, RunRecord{}, saveQueueItem(rootDir, item)
	}
	ready, decision, reasons, err := queueReviewGateAllows(rootDir, item)
	if err != nil {
		return QueueItem{}, RunRecord{}, err
	}
	if !ready {
		item.Status = "manual_required"
		item.Decision = decision
		for _, gateReason := range reasons {
			item.Reasons = appendUniqueQueueReason(item.Reasons, gateReason)
		}
		item.UpdatedAt = now.Format(time.RFC3339Nano)
		return item, RunRecord{}, saveQueueItem(rootDir, item)
	}
	item.AttemptCount++
	run, err := Run(ctx, rootDir, RunOptions{
		Trigger:               firstNonEmptyQueue(item.Trigger, "queue"),
		RequestedBy:           item.RequestedBy,
		IdempotencyKey:        firstNonEmptyQueue(item.IdempotencyKey, item.ID),
		RetryBudget:           item.RetryBudget,
		RetryAttempt:          item.AttemptCount - 1,
		Steps:                 append([]string{}, item.Steps...),
		Environment:           item.Environment,
		ResourceIDs:           append([]string{}, item.ResourceIDs...),
		DeploymentExecutionID: item.DeploymentExecutionID,
	})
	if err != nil {
		return QueueItem{}, RunRecord{}, err
	}
	item.RunID = run.ID
	item.Status = "completed"
	item.Decision = "CONTROL_QUEUE_EXECUTED"
	item.Reasons = appendUniqueQueueReason(item.Reasons, "control_queue_item_executed")
	if run.Status == "manual_required" || run.Status == "failed" {
		item.Status = "manual_required"
		item.Decision = "CONTROL_QUEUE_MANUAL_HANDOFF"
		item.Reasons = appendUniqueQueueReason(item.Reasons, run.Decision)
	}
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveQueueItem(rootDir, item); err != nil {
		return QueueItem{}, RunRecord{}, err
	}
	return item, run, nil
}

func normalizeQueueOptions(options QueueOptions) QueueOptions {
	options.Trigger = normalizeToken(options.Trigger)
	if options.Trigger == "" {
		options.Trigger = "queue"
	}
	options.RequestedBy = strings.TrimSpace(options.RequestedBy)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Environment = normalizeToken(options.Environment)
	options.DeploymentExecutionID = strings.TrimSpace(options.DeploymentExecutionID)
	options.MaintenanceWindow = strings.TrimSpace(options.MaintenanceWindow)
	options.DueAt = strings.TrimSpace(options.DueAt)
	options.AdmissionID = strings.TrimSpace(options.AdmissionID)
	options.RemoteRehearsalID = strings.TrimSpace(options.RemoteRehearsalID)
	options.ReviewPacketID = strings.TrimSpace(options.ReviewPacketID)
	if options.RetryBudget < 0 {
		options.RetryBudget = 0
	}
	options.Steps = normalizeSteps(options.Steps)
	if len(options.Steps) == 0 {
		options.Steps = []string{StepResourceLifecycleScan}
	}
	options.ResourceIDs = compactOptionStrings(options.ResourceIDs)
	return options
}

func normalizeQueueListOptions(options QueueListOptions) QueueListOptions {
	options.Status = normalizeToken(options.Status)
	options.Environment = normalizeToken(options.Environment)
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	return options
}

func normalizeQueueRunOptions(options QueueRunOptions) QueueRunOptions {
	options.Status = normalizeToken(options.Status)
	options.Environment = normalizeToken(options.Environment)
	if options.MaxItems <= 0 {
		options.MaxItems = 5
	}
	if options.MaxItems > 20 {
		options.MaxItems = 20
	}
	return options
}

func queueItemMatches(item QueueItem, options QueueListOptions) bool {
	if options.Status != "" && item.Status != options.Status {
		return false
	}
	if options.Environment != "" && item.Environment != options.Environment {
		return false
	}
	return true
}

func maintenanceWindowAllows(window string, dueAt string, now time.Time) (bool, string) {
	window = strings.TrimSpace(window)
	dueAt = strings.TrimSpace(dueAt)
	if dueAt != "" {
		parsed, err := time.Parse(time.RFC3339, dueAt)
		if err != nil {
			return false, "maintenance_window_invalid"
		}
		if now.Before(parsed.UTC()) {
			return false, "due_at_not_reached"
		}
	}
	if window == "" || window == "always" {
		return true, "maintenance_window_open"
	}
	if strings.HasPrefix(window, "due:") {
		date := strings.TrimPrefix(window, "due:")
		parsed, err := time.Parse("2006-01-02", date)
		if err != nil {
			return false, "maintenance_window_invalid"
		}
		nowDate := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
		if nowDate.Before(parsed.UTC()) {
			return false, "maintenance_window_not_due"
		}
		return true, "maintenance_window_due"
	}
	if strings.HasPrefix(window, "after:") {
		value := strings.TrimPrefix(window, "after:")
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return false, "maintenance_window_invalid"
		}
		if now.Before(parsed.UTC()) {
			return false, "maintenance_window_not_reached"
		}
		return true, "maintenance_window_open"
	}
	if strings.HasPrefix(window, "between:") {
		value := strings.TrimPrefix(window, "between:")
		parts := strings.Split(value, "-")
		if len(parts) != 2 {
			return false, "maintenance_window_invalid"
		}
		start, err := parseHHMM(parts[0])
		if err != nil {
			return false, "maintenance_window_invalid"
		}
		end, err := parseHHMM(parts[1])
		if err != nil {
			return false, "maintenance_window_invalid"
		}
		minute := now.UTC().Hour()*60 + now.UTC().Minute()
		if start <= end {
			if minute < start || minute > end {
				return false, "maintenance_window_closed"
			}
			return true, "maintenance_window_open"
		}
		if minute > end && minute < start {
			return false, "maintenance_window_closed"
		}
		return true, "maintenance_window_open"
	}
	return false, "maintenance_window_invalid"
}

func queueReviewGateAllows(rootDir string, item QueueItem) (bool, string, []string, error) {
	reasons := []string{}
	if item.AdmissionID != "" {
		admission, found, err := findQueueAdmission(rootDir, item.AdmissionID)
		if err != nil {
			return false, "", nil, err
		}
		if !found {
			return false, "CONTROL_QUEUE_REVIEW_GATE_MANUAL_REQUIRED", []string{"write_admission_missing:" + item.AdmissionID}, nil
		}
		if admission.Status != "ready" {
			return false, "CONTROL_QUEUE_REVIEW_GATE_MANUAL_REQUIRED", []string{"write_admission_not_ready:" + admission.Decision}, nil
		}
		reasons = appendUniqueQueueReason(reasons, "write_admission_ready:"+admission.ID)
	}
	if item.RemoteRehearsalID != "" {
		rehearsal, found, err := operations.LoadRemoteExecutionRehearsal(rootDir, item.RemoteRehearsalID)
		if err != nil {
			return false, "", nil, err
		}
		if !found {
			return false, "CONTROL_QUEUE_REVIEW_GATE_MANUAL_REQUIRED", []string{"remote_rehearsal_missing:" + item.RemoteRehearsalID}, nil
		}
		if rehearsal.Status != "completed" {
			return false, "CONTROL_QUEUE_REVIEW_GATE_MANUAL_REQUIRED", []string{"remote_rehearsal_not_completed:" + rehearsal.Decision}, nil
		}
		reasons = appendUniqueQueueReason(reasons, "remote_rehearsal_completed:"+rehearsal.ID)
	}
	if item.ReviewPacketID != "" {
		packet, found, err := operations.LoadWriteReviewPacket(rootDir, item.ReviewPacketID)
		if err != nil {
			return false, "", nil, err
		}
		if !found {
			return false, "CONTROL_QUEUE_REVIEW_GATE_MANUAL_REQUIRED", []string{"write_review_packet_missing:" + item.ReviewPacketID}, nil
		}
		if packet.Status != "ready" {
			return false, "CONTROL_QUEUE_REVIEW_GATE_MANUAL_REQUIRED", []string{"write_review_packet_not_ready:" + packet.Decision}, nil
		}
		reasons = appendUniqueQueueReason(reasons, "write_review_packet_ready:"+packet.ID)
	}
	return true, "CONTROL_QUEUE_REVIEW_GATE_READY", reasons, nil
}

func findQueueAdmission(rootDir string, admissionID string) (operations.WriteAdmissionEntry, bool, error) {
	report, err := operations.BuildWriteAdmissions(rootDir, operations.WriteAdmissionOptions{Limit: 100})
	if err != nil {
		return operations.WriteAdmissionEntry{}, false, err
	}
	for _, admission := range report.Entries {
		if admission.ID == admissionID {
			return admission, true, nil
		}
	}
	return operations.WriteAdmissionEntry{}, false, nil
}

func parseHHMM(value string) (int, error) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func saveQueueItem(rootDir string, item QueueItem) error {
	if err := fsutil.EnsureDir(queueDir(rootDir)); err != nil {
		return err
	}
	if err := fsutil.WriteJSON(filepath.Join(queueDir(rootDir), item.ID+".json"), item); err != nil {
		return err
	}
	return fsutil.AppendJSONL(queueJSONLPath(rootDir), item)
}

func queueDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ControlLoopDir, "queue")
}

func queueJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ControlLoopDir, "queue.jsonl")
}

func appendUniqueQueueReason(values []string, reason string) []string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return values
	}
	for _, existing := range values {
		if existing == reason {
			return values
		}
	}
	return append(values, reason)
}

func firstNonEmptyQueue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
