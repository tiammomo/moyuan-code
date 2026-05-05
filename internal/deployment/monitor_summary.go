package deployment

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type MonitorSummaryOptions struct {
	Environment string `json:"environment,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type MonitorSummary struct {
	ID             string                  `json:"id"`
	Environment    string                  `json:"environment,omitempty"`
	Status         string                  `json:"status"`
	Decision       string                  `json:"decision"`
	Reasons        []string                `json:"reasons"`
	WindowSize     int                     `json:"window_size"`
	HistoryCount   int                     `json:"history_count"`
	FailedCount    int                     `json:"failed_count"`
	BlockedCount   int                     `json:"blocked_count"`
	ManualCount    int                     `json:"manual_count"`
	RollbackCount  int                     `json:"rollback_count"`
	FailureClasses map[string]int          `json:"failure_classes"`
	Latest         []MonitorHistorySummary `json:"latest"`
	EvidenceIDs    []string                `json:"evidence_ids,omitempty"`
	CreatedAt      string                  `json:"created_at"`
}

type MonitorHistorySummary struct {
	ID           string `json:"id"`
	ExecutionID  string `json:"execution_id"`
	DeploymentID string `json:"deployment_id"`
	ReleaseID    string `json:"release_id,omitempty"`
	Environment  string `json:"environment,omitempty"`
	Status       string `json:"status"`
	Decision     string `json:"decision"`
	FailureClass string `json:"failure_class,omitempty"`
	Severity     string `json:"severity,omitempty"`
	Rollback     bool   `json:"rollback"`
	CreatedAt    string `json:"created_at"`
}

func BuildMonitorSummary(rootDir string, options MonitorSummaryOptions) (MonitorSummary, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return MonitorSummary{}, err
	}
	environment := normalizeToken(options.Environment)
	limit := options.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	histories, err := ListPostDeploymentHistories(rootDir, 100)
	if err != nil {
		return MonitorSummary{}, err
	}
	now := time.Now().UTC()
	summary := MonitorSummary{
		ID:             "monitor-summary-" + textutil.Slugify(environmentOrAll(environment)) + "-" + now.Format("20060102150405") + "-" + strconv.FormatInt(now.UnixNano()%1_000_000_000, 10),
		Environment:    environment,
		Status:         "healthy",
		Decision:       "DEPLOYMENT_MONITOR_HEALTHY",
		Reasons:        []string{},
		WindowSize:     limit,
		FailureClasses: map[string]int{},
		Latest:         []MonitorHistorySummary{},
		EvidenceIDs:    []string{},
		CreatedAt:      now.Format(time.RFC3339Nano),
	}
	for _, history := range histories {
		if environment != "" && history.Environment != environment {
			continue
		}
		if len(summary.Latest) >= limit {
			continue
		}
		summary.HistoryCount++
		summary.Latest = append(summary.Latest, monitorHistorySummary(history))
		summary.EvidenceIDs = append(summary.EvidenceIDs, history.EvidenceIDs...)
		switch history.Status {
		case "failed":
			summary.FailedCount++
		case "blocked":
			summary.BlockedCount++
		case "manual_required":
			summary.ManualCount++
		}
		if history.Rollback.Required {
			summary.RollbackCount++
		}
		if history.FailureClass != "" && history.FailureClass != "none" {
			summary.FailureClasses[history.FailureClass]++
		}
	}
	summary.Reasons = monitorSummaryReasons(summary)
	summary.Status, summary.Decision = monitorSummaryDecision(summary, environment)
	return finishMonitorSummary(rootDir, summary)
}

func LoadMonitorSummary(rootDir string, id string) (MonitorSummary, bool, error) {
	var summary MonitorSummary
	found, err := fsutil.ReadJSON(monitorSummaryPath(rootDir, id), &summary)
	return summary, found, err
}

func ListMonitorSummaries(rootDir string, limit int) ([]MonitorSummary, error) {
	if err := fsutil.EnsureDir(monitorSummaryDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(monitorSummaryDir(rootDir))
	if err != nil {
		return nil, err
	}
	summaries := []MonitorSummary{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var summary MonitorSummary
		found, err := fsutil.ReadJSON(filepath.Join(monitorSummaryDir(rootDir), entry.Name()), &summary)
		if err != nil {
			return nil, err
		}
		if found && summary.ID != "" {
			summaries = append(summaries, summary)
		}
	}
	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].CreatedAt > summaries[j].CreatedAt
	})
	if limit <= 0 {
		limit = 20
	}
	if limit > 0 && len(summaries) > limit {
		return summaries[:limit], nil
	}
	return summaries, nil
}

func finishMonitorSummary(rootDir string, summary MonitorSummary) (MonitorSummary, error) {
	if err := fsutil.EnsureDir(monitorSummaryDir(rootDir)); err != nil {
		return MonitorSummary{}, err
	}
	if err := fsutil.WriteJSON(monitorSummaryPath(rootDir, summary.ID), summary); err != nil {
		return MonitorSummary{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "monitor-summaries.jsonl"), summary); err != nil {
		return MonitorSummary{}, err
	}
	_ = logging.Log(rootDir, "release", "deployment.monitor.summary.created", map[string]any{
		"monitor_summary_id": summary.ID,
		"environment":        summary.Environment,
		"decision":           summary.Decision,
		"status":             summary.Status,
		"history_count":      summary.HistoryCount,
		"failed_count":       summary.FailedCount,
	})
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "deployment_monitor_summary",
		ParentID:    summary.ID,
		SubjectType: "deployment",
		SubjectID:   environmentOrAll(summary.Environment),
		Operation:   "deployment.monitor.summary",
		Status:      summary.Status,
		Decision:    summary.Decision,
		Reasons:     summary.Reasons,
		Source:      "deployment",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "deployment_monitor_summary",
			ID:   summary.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "monitor-summaries", summary.ID+".json")),
		}},
	}); err != nil {
		return MonitorSummary{}, err
	}
	return summary, nil
}

func monitorHistorySummary(history PostDeploymentHistory) MonitorHistorySummary {
	return MonitorHistorySummary{
		ID:           history.ID,
		ExecutionID:  history.ExecutionID,
		DeploymentID: history.DeploymentID,
		ReleaseID:    history.ReleaseID,
		Environment:  history.Environment,
		Status:       history.Status,
		Decision:     history.Decision,
		FailureClass: history.FailureClass,
		Severity:     history.Severity,
		Rollback:     history.Rollback.Required,
		CreatedAt:    history.CreatedAt,
	}
}

func monitorSummaryReasons(summary MonitorSummary) []string {
	reasons := []string{}
	if summary.HistoryCount == 0 {
		reasons = append(reasons, "monitor_history_empty")
		return reasons
	}
	reasons = append(reasons, "history_count:"+strconv.Itoa(summary.HistoryCount))
	if summary.FailedCount > 0 {
		reasons = append(reasons, "failed_count:"+strconv.Itoa(summary.FailedCount))
	}
	if summary.BlockedCount > 0 {
		reasons = append(reasons, "blocked_count:"+strconv.Itoa(summary.BlockedCount))
	}
	if summary.ManualCount > 0 {
		reasons = append(reasons, "manual_count:"+strconv.Itoa(summary.ManualCount))
	}
	if summary.RollbackCount > 0 {
		reasons = append(reasons, "rollback_count:"+strconv.Itoa(summary.RollbackCount))
	}
	for failureClass, count := range summary.FailureClasses {
		reasons = append(reasons, "failure_class:"+failureClass+":"+strconv.Itoa(count))
	}
	sort.Strings(reasons)
	return reasons
}

func monitorSummaryDecision(summary MonitorSummary, environment string) (string, string) {
	if summary.HistoryCount == 0 {
		return "unknown", "DEPLOYMENT_MONITOR_NO_HISTORY"
	}
	if summary.FailedCount > 0 || summary.RollbackCount > 0 {
		if environment == "production" {
			return "critical", "PRODUCTION_MONITOR_CRITICAL"
		}
		return "attention_required", "DEPLOYMENT_MONITOR_ATTENTION_REQUIRED"
	}
	if summary.BlockedCount > 0 || summary.ManualCount > 0 {
		return "attention_required", "DEPLOYMENT_MONITOR_ATTENTION_REQUIRED"
	}
	return "healthy", "DEPLOYMENT_MONITOR_HEALTHY"
}

func monitorSummaryDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "monitor-summaries")
}

func monitorSummaryPath(rootDir string, id string) string {
	return filepath.Join(monitorSummaryDir(rootDir), id+".json")
}

func environmentOrAll(environment string) string {
	if strings.TrimSpace(environment) == "" {
		return "all"
	}
	return environment
}
