package skills

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type Effectiveness struct {
	ID              string   `json:"id"`
	SkillID         string   `json:"skill_id"`
	BindingID       string   `json:"binding_id,omitempty"`
	SubagentID      string   `json:"subagent_id,omitempty"`
	RunID           string   `json:"run_id,omitempty"`
	IssueID         string   `json:"issue_id,omitempty"`
	Outcome         string   `json:"outcome"`
	QualityImpact   string   `json:"quality_impact"`
	ReworkReduced   bool     `json:"rework_reduced"`
	DurationSeconds int      `json:"duration_seconds,omitempty"`
	Findings        []string `json:"findings,omitempty"`
	CreatedAt       string   `json:"created_at"`
}

func RecordEffectiveness(rootDir string, record Effectiveness) (Effectiveness, error) {
	record, err := normalizeEffectivenessForSave(rootDir, record)
	if err != nil {
		return Effectiveness{}, err
	}
	if err := fsutil.WriteJSON(effectivenessPath(rootDir, record.ID), record); err != nil {
		return Effectiveness{}, err
	}
	if err := fsutil.AppendJSONL(effectivenessEventsPath(rootDir), record); err != nil {
		return Effectiveness{}, err
	}
	_ = logging.Log(rootDir, "audit", "skill.effectiveness.recorded", map[string]any{"effectiveness_id": record.ID, "skill_id": record.SkillID, "outcome": record.Outcome, "quality_impact": record.QualityImpact})
	return record, nil
}

func ListEffectiveness(rootDir string, skillID string, limit int) ([]Effectiveness, error) {
	if err := fsutil.EnsureDir(effectivenessDir(rootDir)); err != nil {
		return nil, err
	}
	lines, err := fsutil.TailLines(effectivenessEventsPath(rootDir), limit*3)
	if err != nil {
		return nil, err
	}
	skillID = normalizeID(skillID)
	seen := map[string]bool{}
	records := []Effectiveness{}
	for idx := len(lines) - 1; idx >= 0; idx-- {
		var record Effectiveness
		if err := json.Unmarshal([]byte(lines[idx]), &record); err != nil {
			continue
		}
		if record.ID == "" || seen[record.ID] {
			continue
		}
		if skillID != "" && record.SkillID != skillID {
			continue
		}
		seen[record.ID] = true
		records = append(records, record)
		if limit > 0 && len(records) >= limit {
			break
		}
	}
	sort.SliceStable(records, func(i, j int) bool { return records[i].CreatedAt > records[j].CreatedAt })
	return records, nil
}

func normalizeEffectivenessForSave(rootDir string, record Effectiveness) (Effectiveness, error) {
	record.SkillID = normalizeID(record.SkillID)
	record.BindingID = normalizeID(record.BindingID)
	record.SubagentID = strings.TrimSpace(record.SubagentID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.IssueID = strings.TrimSpace(record.IssueID)
	record.Outcome = normalizeToken(record.Outcome)
	record.QualityImpact = normalizeToken(record.QualityImpact)
	record.Findings = normalizeFindingTexts(record.Findings)
	if record.SkillID == "" {
		return Effectiveness{}, errors.New("skill_id_required")
	}
	if _, found, err := Show(rootDir, record.SkillID); err != nil {
		return Effectiveness{}, err
	} else if !found {
		return Effectiveness{}, errors.New("skill_not_found")
	}
	if record.SubagentID == "" && record.RunID == "" && record.IssueID == "" {
		return Effectiveness{}, errors.New("effectiveness_reference_required")
	}
	if record.Outcome == "" {
		record.Outcome = "neutral"
	}
	if !allowedOutcome(record.Outcome) {
		return Effectiveness{}, errors.New("effectiveness_outcome_invalid")
	}
	if record.QualityImpact == "" {
		record.QualityImpact = "unchanged"
	}
	if !allowedQualityImpact(record.QualityImpact) {
		return Effectiveness{}, errors.New("quality_impact_invalid")
	}
	if record.ID == "" {
		record.ID = "skill-eff-" + record.SkillID + "-" + time.Now().UTC().Format("20060102150405")
	}
	record.ID = normalizeID(record.ID)
	record.CreatedAt = now()
	return record, nil
}

func normalizeFindingTexts(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		if containsPlainSecret(value) {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func allowedOutcome(value string) bool {
	return value == "helped" || value == "neutral" || value == "harmful" || value == "blocked"
}

func allowedQualityImpact(value string) bool {
	return value == "improved" || value == "unchanged" || value == "worsened"
}

func effectivenessDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).SkillsDir, "effectiveness")
}

func effectivenessPath(rootDir string, id string) string {
	return filepath.Join(effectivenessDir(rootDir), id+".json")
}

func effectivenessEventsPath(rootDir string) string {
	return filepath.Join(effectivenessDir(rootDir), "effectiveness.jsonl")
}
