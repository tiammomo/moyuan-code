package skills

import (
	"path/filepath"
	"sort"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type RecommendOptions struct {
	IssueID   string `json:"issue_id,omitempty"`
	Role      string `json:"role"`
	TaskType  string `json:"task_type,omitempty"`
	RiskLevel string `json:"risk_level,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type RecommendationReport struct {
	ID         string                    `json:"id"`
	IssueID    string                    `json:"issue_id,omitempty"`
	Role       string                    `json:"role"`
	TaskType   string                    `json:"task_type,omitempty"`
	RiskLevel  string                    `json:"risk_level"`
	Candidates []RecommendationCandidate `json:"candidates"`
	CreatedAt  string                    `json:"created_at"`
}

type RecommendationCandidate struct {
	SkillID string   `json:"skill_id"`
	Name    string   `json:"name"`
	Source  string   `json:"source"`
	Score   float64  `json:"score"`
	Reasons []string `json:"reasons"`
	Risks   []string `json:"risks,omitempty"`
}

func Recommend(rootDir string, options RecommendOptions) (RecommendationReport, error) {
	skills, err := List(rootDir)
	if err != nil {
		return RecommendationReport{}, err
	}
	options.Role = normalizeToken(options.Role)
	options.TaskType = normalizeToken(options.TaskType)
	options.RiskLevel = normalizeToken(options.RiskLevel)
	if options.Role == "" {
		options.Role = "backend"
	}
	if options.RiskLevel == "" {
		options.RiskLevel = "medium"
	}
	if options.Limit <= 0 {
		options.Limit = 5
	}
	report := RecommendationReport{
		ID:         "skill-rec-" + time.Now().UTC().Format("20060102150405"),
		IssueID:    options.IssueID,
		Role:       options.Role,
		TaskType:   options.TaskType,
		RiskLevel:  options.RiskLevel,
		Candidates: []RecommendationCandidate{},
		CreatedAt:  now(),
	}
	for _, skill := range skills {
		if !skill.Enabled {
			continue
		}
		candidate, ok := scoreCandidate(rootDir, skill, options)
		if ok {
			report.Candidates = append(report.Candidates, candidate)
		}
	}
	sort.SliceStable(report.Candidates, func(i, j int) bool {
		if report.Candidates[i].Score == report.Candidates[j].Score {
			return report.Candidates[i].SkillID < report.Candidates[j].SkillID
		}
		return report.Candidates[i].Score > report.Candidates[j].Score
	})
	if len(report.Candidates) > options.Limit {
		report.Candidates = report.Candidates[:options.Limit]
	}
	if err := fsutil.AppendJSONL(recommendationsPath(rootDir), report); err != nil {
		return RecommendationReport{}, err
	}
	_ = logging.Log(rootDir, "audit", "skill.recommendation.created", map[string]any{"recommendation_id": report.ID, "role": report.Role, "task_type": report.TaskType, "candidates": len(report.Candidates)})
	return report, nil
}

func scoreCandidate(rootDir string, skill Definition, options RecommendOptions) (RecommendationCandidate, bool) {
	score := 0.35
	reasons := []string{"enabled_skill"}
	risks := []string{}
	if len(skill.CompatibleRoles) == 0 {
		score += 0.08
		reasons = append(reasons, "generic_role")
	} else if contains(skill.CompatibleRoles, options.Role) {
		score += 0.28
		reasons = append(reasons, "role_match:"+options.Role)
	} else {
		return RecommendationCandidate{}, false
	}
	if options.TaskType != "" {
		if contains(skill.Tags, options.TaskType) || contains(skill.RequiredTools, options.TaskType) {
			score += 0.22
			reasons = append(reasons, "task_match:"+options.TaskType)
		}
	}
	switch {
	case options.RiskLevel == "low" && skill.RiskLevel == "high":
		score -= 0.25
		risks = append(risks, "high_risk_skill_for_low_risk_task")
	case options.RiskLevel == "high" && skill.RiskLevel == "low":
		score += 0.08
		reasons = append(reasons, "low_risk_skill_for_high_risk_task")
	default:
		score += 0.1
		reasons = append(reasons, "risk_fit:"+skill.RiskLevel)
	}
	if len(skill.RequiredTools) > 0 {
		reasons = append(reasons, "requires_tools")
	}
	adjustment, effectivenessReasons, effectivenessRisks := effectivenessAdjustment(rootDir, skill.ID)
	score += adjustment
	reasons = append(reasons, effectivenessReasons...)
	risks = append(risks, effectivenessRisks...)
	if score <= 0 {
		return RecommendationCandidate{}, false
	}
	if score > 1 {
		score = 1
	}
	return RecommendationCandidate{
		SkillID: skill.ID,
		Name:    skill.Name,
		Source:  skill.Source,
		Score:   roundScore(score),
		Reasons: reasons,
		Risks:   risks,
	}, true
}

func effectivenessAdjustment(rootDir string, skillID string) (float64, []string, []string) {
	records, err := ListEffectiveness(rootDir, skillID, 20)
	if err != nil || len(records) == 0 {
		return 0, []string{}, []string{}
	}
	helped := 0
	improved := 0
	reworkReduced := 0
	harmful := 0
	worsened := 0
	blocked := 0
	for _, record := range records {
		switch record.Outcome {
		case "helped":
			helped++
		case "harmful":
			harmful++
		case "blocked":
			blocked++
		}
		switch record.QualityImpact {
		case "improved":
			improved++
		case "worsened":
			worsened++
		}
		if record.ReworkReduced {
			reworkReduced++
		}
	}
	score := float64(helped)*0.04 + float64(improved)*0.04 + float64(reworkReduced)*0.03 - float64(harmful)*0.12 - float64(worsened)*0.1 - float64(blocked)*0.05
	if score > 0.18 {
		score = 0.18
	}
	if score < -0.3 {
		score = -0.3
	}
	reasons := []string{}
	risks := []string{}
	if helped > 0 {
		reasons = append(reasons, "effectiveness_helped")
	}
	if improved > 0 {
		reasons = append(reasons, "quality_improved")
	}
	if reworkReduced > 0 {
		reasons = append(reasons, "rework_reduced")
	}
	if harmful > 0 {
		risks = append(risks, "effectiveness_harmful")
	}
	if worsened > 0 {
		risks = append(risks, "quality_worsened")
	}
	if blocked > 0 {
		risks = append(risks, "effectiveness_blocked")
	}
	return score, reasons, risks
}

func recommendationsPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).SkillsDir, "recommendations.jsonl")
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func roundScore(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
