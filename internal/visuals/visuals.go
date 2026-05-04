package visuals

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/process"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type DiagramOptions struct {
	DiagramType string `json:"diagram_type"`
	Title       string `json:"title"`
	Scope       string `json:"scope,omitempty"`
	Size        string `json:"size,omitempty"`
}

type DiagramSpec struct {
	ID          string        `json:"id"`
	DiagramType string        `json:"diagram_type"`
	Title       string        `json:"title"`
	Scope       string        `json:"scope,omitempty"`
	Nodes       []DiagramNode `json:"nodes"`
	Edges       []DiagramEdge `json:"edges"`
	Safety      SafetyPolicy  `json:"safety"`
	PromptPath  string        `json:"prompt_path,omitempty"`
	SpecPath    string        `json:"spec_path,omitempty"`
	CreatedAt   string        `json:"created_at"`
}

type DiagramNode struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Kind    string   `json:"kind"`
	Details []string `json:"details"`
}

type DiagramEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
}

type SafetyPolicy struct {
	StripSecrets      bool     `json:"strip_secrets"`
	StripPrivateIPs   bool     `json:"strip_private_ips"`
	ForbiddenPatterns []string `json:"forbidden_patterns"`
}

type AssetRecord struct {
	ID              string                  `json:"id"`
	DiagramSpecID   string                  `json:"diagram_spec_id"`
	DiagramType     string                  `json:"diagram_type"`
	Title           string                  `json:"title"`
	Status          string                  `json:"status"`
	ProviderID      string                  `json:"provider_id,omitempty"`
	ModelID         string                  `json:"model_id,omitempty"`
	RouteDecision   providers.RouteDecision `json:"route_decision"`
	Size            string                  `json:"size"`
	ImagePath       string                  `json:"image_path,omitempty"`
	PromptPath      string                  `json:"prompt_path"`
	SpecPath        string                  `json:"spec_path"`
	ExplanationPath string                  `json:"explanation_path,omitempty"`
	CreatedAt       string                  `json:"created_at"`
	UpdatedAt       string                  `json:"updated_at"`
}

type Plan struct {
	DiagramSpec DiagramSpec `json:"diagram_spec"`
	Asset       AssetRecord `json:"asset"`
}

type RenderOptions struct {
	AssetID  string `json:"asset_id"`
	Mode     string `json:"mode"`
	Approved bool   `json:"approved"`
}

type RenderExecution struct {
	ID               string         `json:"id"`
	AssetID          string         `json:"asset_id"`
	DiagramSpecID    string         `json:"diagram_spec_id,omitempty"`
	DiagramType      string         `json:"diagram_type,omitempty"`
	Title            string         `json:"title,omitempty"`
	Mode             string         `json:"mode"`
	Status           string         `json:"status"`
	Decision         string         `json:"decision"`
	Reasons          []string       `json:"reasons"`
	ProviderID       string         `json:"provider_id,omitempty"`
	ModelID          string         `json:"model_id,omitempty"`
	Size             string         `json:"size,omitempty"`
	PromptPath       string         `json:"prompt_path,omitempty"`
	SpecPath         string         `json:"spec_path,omitempty"`
	ImagePath        string         `json:"image_path,omitempty"`
	ScriptPath       string         `json:"script_path,omitempty"`
	AuthRef          string         `json:"auth_ref,omitempty"`
	EnvKeys          []string       `json:"env_keys,omitempty"`
	ApprovalID       string         `json:"approval_id,omitempty"`
	Quality          *RenderQuality `json:"quality,omitempty"`
	PreviewIndexPath string         `json:"preview_index_path,omitempty"`
	Steps            []RenderStep   `json:"steps"`
	StartedAt        string         `json:"started_at"`
	FinishedAt       string         `json:"finished_at,omitempty"`
}

type RenderStep struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Command    string `json:"command,omitempty"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
}

type RenderQuality struct {
	Status    string         `json:"status"`
	Checks    []QualityCheck `json:"checks"`
	CheckedAt string         `json:"checked_at"`
}

type QualityCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type PreviewIndexRecord struct {
	ExecutionID     string `json:"execution_id"`
	AssetID         string `json:"asset_id"`
	ProviderID      string `json:"provider_id,omitempty"`
	ModelID         string `json:"model_id,omitempty"`
	ImagePath       string `json:"image_path,omitempty"`
	ExplanationPath string `json:"explanation_path,omitempty"`
	QualityStatus   string `json:"quality_status"`
	CreatedAt       string `json:"created_at"`
}

type scriptAuth struct {
	AuthRef string
	APIKey  string
	BaseURL string
}

var (
	credentialPattern = regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret)\s*[:=]\s*[^,\s]+`)
	openAIKeyPattern  = regexp.MustCompile(`sk-[A-Za-z0-9_-]{12,}`)
	privateIPPattern  = regexp.MustCompile(`\b(10\.\d{1,3}\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2\d|3[0-1])\.\d{1,3}\.\d{1,3})\b`)
)

func GeneratePlan(rootDir string, options DiagramOptions) (Plan, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Plan{}, err
	}
	diagramType := normalizeDiagramType(options.DiagramType)
	if diagramType == "" {
		return Plan{}, errors.New("diagram_type_required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	id := "diagram-" + textutil.Slugify(diagramType+"-"+time.Now().UTC().Format("20060102150405"))
	title := strings.TrimSpace(options.Title)
	if title == "" {
		title = defaultTitle(diagramType)
	}
	spec := DiagramSpec{
		ID:          id,
		DiagramType: diagramType,
		Title:       sanitize(options.TitleOr(title)),
		Scope:       sanitize(options.Scope),
		Nodes:       nodesFor(diagramType),
		Edges:       edgesFor(diagramType),
		Safety: SafetyPolicy{
			StripSecrets:      true,
			StripPrivateIPs:   true,
			ForbiddenPatterns: []string{"api_key", "token", "password", "secret", "private_ip"},
		},
		CreatedAt: now,
	}
	prompt := buildPrompt(spec)
	specPath := filepath.Join(visualsDir(rootDir), "specs", spec.ID+".json")
	promptPath := filepath.Join(visualsDir(rootDir), "prompts", spec.ID+".prompt.md")
	spec.SpecPath = specPath
	spec.PromptPath = promptPath
	if err := fsutil.WriteJSON(specPath, spec); err != nil {
		return Plan{}, err
	}
	if err := fsutil.WriteText(promptPath, prompt); err != nil {
		return Plan{}, err
	}
	route, err := providers.Route(rootDir, providers.RouteRequest{ModelStrategy: "image-diagram"})
	if err != nil {
		return Plan{}, err
	}
	size := strings.TrimSpace(options.Size)
	if size == "" {
		size = "3072x2048"
	}
	status := "planned"
	if route.Blocked {
		status = "route_blocked"
	}
	asset := AssetRecord{
		ID:            "visual-" + textutil.Slugify(spec.ID),
		DiagramSpecID: spec.ID,
		DiagramType:   diagramType,
		Title:         spec.Title,
		Status:        status,
		ProviderID:    route.ProviderID,
		ModelID:       route.ModelID,
		RouteDecision: route,
		Size:          size,
		PromptPath:    promptPath,
		SpecPath:      specPath,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := writeAsset(rootDir, asset); err != nil {
		return Plan{}, err
	}
	_ = logging.Log(rootDir, "model", "visual.diagram.planned", map[string]any{"asset_id": asset.ID, "diagram_spec_id": spec.ID, "status": asset.Status, "provider_id": asset.ProviderID})
	return Plan{DiagramSpec: spec, Asset: asset}, nil
}

func ListAssets(rootDir string, limit int) ([]AssetRecord, error) {
	dir := assetsDir(rootDir)
	if err := fsutil.EnsureDir(dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	assets := []AssetRecord{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var asset AssetRecord
		found, err := fsutil.ReadJSON(filepath.Join(dir, entry.Name()), &asset)
		if err != nil {
			return nil, err
		}
		if found && asset.ID != "" {
			assets = append(assets, asset)
		}
	}
	sort.SliceStable(assets, func(i, j int) bool {
		return assets[i].UpdatedAt > assets[j].UpdatedAt
	})
	if limit > 0 && len(assets) > limit {
		return assets[:limit], nil
	}
	return assets, nil
}

func LoadAsset(rootDir string, id string) (AssetRecord, bool, error) {
	var asset AssetRecord
	found, err := fsutil.ReadJSON(filepath.Join(assetsDir(rootDir), textutil.Slugify(id)+".json"), &asset)
	return asset, found, err
}

func RenderAsset(ctx context.Context, rootDir string, options RenderOptions) (RenderExecution, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return RenderExecution{}, err
	}
	options.AssetID = strings.TrimSpace(options.AssetID)
	options.Mode = normalizeRenderMode(options.Mode)
	if options.AssetID == "" {
		return RenderExecution{}, errors.New("asset_id_required")
	}
	if options.Mode == "" {
		options.Mode = "dry_run"
	}
	now := time.Now().UTC()
	asset, found, err := LoadAsset(rootDir, options.AssetID)
	if err != nil {
		return RenderExecution{}, err
	}
	execution := RenderExecution{
		ID:            "visual-render-" + textutil.Slugify(options.AssetID+"-"+options.Mode) + "-" + now.Format("20060102150405"),
		AssetID:       options.AssetID,
		DiagramSpecID: asset.DiagramSpecID,
		DiagramType:   asset.DiagramType,
		Title:         asset.Title,
		Mode:          options.Mode,
		Status:        "blocked",
		Decision:      "VISUAL_RENDER_BLOCKED",
		Reasons:       []string{},
		ProviderID:    asset.ProviderID,
		ModelID:       asset.ModelID,
		Size:          asset.Size,
		PromptPath:    asset.PromptPath,
		SpecPath:      asset.SpecPath,
		ImagePath:     asset.ImagePath,
		Steps:         []RenderStep{},
		StartedAt:     now.Format(time.RFC3339Nano),
	}
	if !found {
		execution.Reasons = append(execution.Reasons, "visual_asset_not_found")
		return finishRenderExecution(rootDir, execution)
	}
	scriptPath := scriptFor(asset.DiagramType)
	execution.ScriptPath = scriptPath
	switch options.Mode {
	case "dry_run":
		execution.Status = "completed"
		execution.Decision = "VISUAL_RENDER_DRY_RUN"
		execution.Reasons = append(execution.Reasons, "no_image_api_called")
		if asset.Status == "route_blocked" {
			execution.Reasons = append(execution.Reasons, "visual_asset_route_blocked_preview_only")
		}
		execution.Steps = append(execution.Steps, RenderStep{
			Name:    "script_preview",
			Status:  "dry_run",
			Command: renderCommand(scriptPath),
			Output:  "script execution preview only",
		})
	case "script":
		if !options.Approved {
			execution.Reasons = append(execution.Reasons, "visual_render_approval_required")
			approval, err := approvals.Request(rootDir, approvals.RequestOptions{
				TargetType:  "visual_render",
				TargetID:    execution.ID,
				Action:      "visual.render.script",
				RiskLevel:   "high",
				RequestedBy: "system",
				Reason:      "script render requires approval before image API credential injection",
				Metadata: map[string]any{
					"asset_id":    execution.AssetID,
					"provider_id": execution.ProviderID,
					"model_id":    execution.ModelID,
				},
			})
			if err != nil {
				return RenderExecution{}, err
			}
			execution.ApprovalID = approval.ID
			return finishRenderExecution(rootDir, execution)
		}
		if asset.Status == "route_blocked" {
			execution.Reasons = append(execution.Reasons, "visual_asset_route_blocked")
			return finishRenderExecution(rootDir, execution)
		}
		if os.Getenv("MOYUAN_ALLOW_IMAGE_SCRIPT") != "1" {
			execution.Reasons = append(execution.Reasons, "image_script_execution_not_enabled")
			return finishRenderExecution(rootDir, execution)
		}
		auth, authReason, err := resolveScriptAuth(rootDir, asset.ProviderID)
		if err != nil {
			return RenderExecution{}, err
		}
		if authReason != "" {
			execution.Reasons = append(execution.Reasons, authReason)
			return finishRenderExecution(rootDir, execution)
		}
		execution.AuthRef = auth.AuthRef
		if !fsutil.Exists(filepath.Join(rootDir, scriptPath)) {
			execution.Reasons = append(execution.Reasons, "visual_render_script_missing")
			return finishRenderExecution(rootDir, execution)
		}
		execution = runScript(ctx, rootDir, execution, asset, scriptPath, auth)
	default:
		execution.Reasons = append(execution.Reasons, "visual_render_mode_not_allowed:"+options.Mode)
	}
	return finishRenderExecution(rootDir, execution)
}

func LoadRenderExecution(rootDir string, id string) (RenderExecution, bool, error) {
	var execution RenderExecution
	found, err := fsutil.ReadJSON(filepath.Join(renderExecutionsDir(rootDir), textutil.Slugify(id)+".json"), &execution)
	return execution, found, err
}

func ListRenderExecutions(rootDir string, limit int) ([]RenderExecution, error) {
	dir := renderExecutionsDir(rootDir)
	if err := fsutil.EnsureDir(dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	executions := []RenderExecution{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var execution RenderExecution
		found, err := fsutil.ReadJSON(filepath.Join(dir, entry.Name()), &execution)
		if err != nil {
			return nil, err
		}
		if found && execution.ID != "" {
			executions = append(executions, execution)
		}
	}
	sort.SliceStable(executions, func(i, j int) bool {
		return executions[i].StartedAt > executions[j].StartedAt
	})
	if limit > 0 && len(executions) > limit {
		return executions[:limit], nil
	}
	return executions, nil
}

func writeAsset(rootDir string, asset AssetRecord) error {
	if err := fsutil.WriteJSON(filepath.Join(assetsDir(rootDir), asset.ID+".json"), asset); err != nil {
		return err
	}
	return fsutil.AppendJSONL(filepath.Join(assetsDir(rootDir), "assets.jsonl"), asset)
}

func resolveScriptAuth(rootDir string, providerID string) (scriptAuth, string, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return scriptAuth{}, "visual_provider_required", nil
	}
	provider, found, err := providers.Show(rootDir, providerID)
	if err != nil {
		return scriptAuth{}, "", err
	}
	if !found {
		return scriptAuth{}, "visual_provider_not_found:" + providerID, nil
	}
	if !provider.Enabled {
		return scriptAuth{}, "visual_provider_disabled:" + provider.ID, nil
	}
	authRef := strings.TrimSpace(provider.AuthRef)
	if authRef == "" {
		return scriptAuth{}, "image_provider_auth_ref_missing", nil
	}
	if strings.HasPrefix(authRef, "secret:") {
		return scriptAuth{}, "image_provider_secret_ref_not_supported_for_script", nil
	}
	if !strings.HasPrefix(authRef, "env:") {
		return scriptAuth{}, "image_provider_auth_ref_unsupported", nil
	}
	key := strings.TrimSpace(strings.TrimPrefix(authRef, "env:"))
	if key == "" {
		return scriptAuth{}, "image_provider_auth_env_required", nil
	}
	value := os.Getenv(key)
	if value == "" {
		return scriptAuth{}, "image_provider_auth_env_missing:" + key, nil
	}
	return scriptAuth{AuthRef: authRef, APIKey: value, BaseURL: strings.TrimSpace(provider.BaseURL)}, "", nil
}

func runScript(ctx context.Context, rootDir string, execution RenderExecution, asset AssetRecord, scriptPath string, auth scriptAuth) RenderExecution {
	step := RenderStep{Name: "image_script", Status: "running", Command: renderCommand(scriptPath), StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	env := filterEnvKeys(os.Environ(), "IMAGE_SIZE", "OPENAI_API_KEY", "OPENAI_IMAGE_MODEL", "OPENAI_BASE_URL")
	env = append(env, "IMAGE_SIZE="+asset.Size, "OPENAI_API_KEY="+auth.APIKey)
	envKeys := []string{"IMAGE_SIZE", "OPENAI_API_KEY"}
	if asset.ModelID != "" {
		env = append(env, "OPENAI_IMAGE_MODEL="+asset.ModelID)
		envKeys = append(envKeys, "OPENAI_IMAGE_MODEL")
	}
	if auth.BaseURL != "" {
		env = append(env, "OPENAI_BASE_URL="+auth.BaseURL)
		envKeys = append(envKeys, "OPENAI_BASE_URL")
	}
	execution.EnvKeys = envKeys
	result := process.RunCommandInput(ctx, rootDir, "", env, "node", scriptPath)
	rawStdout := result.Stdout
	step.Output = sanitizeExecutionText(rawStdout)
	step.Error = sanitizeExecutionText(result.Stderr)
	step.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if result.Code == 0 {
		imagePath := outputPathFromScript(rawStdout, "Image written:")
		if imagePath != "" {
			asset.ImagePath = imagePath
			execution.ImagePath = imagePath
		}
		promptPath := outputPathFromScript(rawStdout, "Prompt written:")
		if promptPath != "" {
			asset.PromptPath = promptPath
			execution.PromptPath = promptPath
		}
		explanationPath := outputPathFromScript(rawStdout, "Explanation written:")
		if explanationPath != "" {
			asset.ExplanationPath = explanationPath
		}
		quality := evaluateRenderQuality(rootDir, execution, asset, step)
		execution.Quality = &quality
		if quality.Status == "passed" {
			step.Status = "completed"
			execution.Status = "completed"
			execution.Decision = "VISUAL_RENDER_COMPLETED"
			execution.Reasons = append(execution.Reasons, "image_script_completed", "image_quality_passed")
			asset.Status = "rendered"
			asset.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			if previewPath, err := writePreviewIndex(rootDir, execution, asset, quality); err == nil {
				execution.PreviewIndexPath = previewPath
			}
			_ = writeAsset(rootDir, asset)
		} else {
			step.Status = "failed"
			execution.Status = "failed"
			execution.Decision = "VISUAL_RENDER_QUALITY_FAILED"
			execution.Reasons = append(execution.Reasons, "image_quality_failed")
			asset.Status = "quality_failed"
			asset.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			_ = writeAsset(rootDir, asset)
		}
	} else {
		step.Status = "failed"
		execution.Status = "failed"
		execution.Decision = "VISUAL_RENDER_FAILED"
		execution.Reasons = append(execution.Reasons, "image_script_failed")
	}
	execution.Steps = append(execution.Steps, step)
	return execution
}

func evaluateRenderQuality(rootDir string, execution RenderExecution, asset AssetRecord, step RenderStep) RenderQuality {
	quality := RenderQuality{Status: "passed", Checks: []QualityCheck{}, CheckedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	add := func(name string, passed bool, reason string) {
		status := "passed"
		if !passed {
			status = "failed"
			quality.Status = "failed"
		}
		quality.Checks = append(quality.Checks, QualityCheck{Name: name, Status: status, Reason: reason})
	}
	imagePath, imageOK := pathWithinRoot(rootDir, execution.ImagePath)
	add("image_path_present", execution.ImagePath != "", "image_path_required")
	add("image_path_within_root", imageOK, "image_path_must_stay_inside_project")
	add("image_file_exists", imageOK && fsutil.Exists(imagePath), "image_file_missing")
	add("image_format_supported", hasSupportedImageExtension(execution.ImagePath), "image_format_must_be_png_jpg_or_webp")
	promptPath, promptOK := pathWithinRoot(rootDir, execution.PromptPath)
	add("prompt_file_exists", promptOK && fsutil.Exists(promptPath), "prompt_file_missing")
	specPath, specOK := pathWithinRoot(rootDir, execution.SpecPath)
	add("spec_file_exists", specOK && fsutil.Exists(specPath), "spec_file_missing")
	explanationPath, explanationOK := pathWithinRoot(rootDir, asset.ExplanationPath)
	add("explanation_path_present", asset.ExplanationPath != "", "explanation_path_required")
	add("explanation_file_exists", explanationOK && fsutil.Exists(explanationPath), "explanation_file_missing")
	add("script_output_sanitized", !containsSensitiveText(step.Output) && !containsSensitiveText(step.Error), "script_output_contains_sensitive_text")
	return quality
}

func writePreviewIndex(rootDir string, execution RenderExecution, asset AssetRecord, quality RenderQuality) (string, error) {
	path := filepath.Join(previewsDir(rootDir), "index.jsonl")
	err := fsutil.AppendJSONL(path, PreviewIndexRecord{
		ExecutionID:     execution.ID,
		AssetID:         execution.AssetID,
		ProviderID:      execution.ProviderID,
		ModelID:         execution.ModelID,
		ImagePath:       execution.ImagePath,
		ExplanationPath: asset.ExplanationPath,
		QualityStatus:   quality.Status,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339Nano),
	})
	return path, err
}

func pathWithinRoot(rootDir string, candidate string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", false
	}
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(rootDir, candidate)
	}
	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return "", false
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(rootAbs, candidateAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return candidateAbs, false
	}
	return candidateAbs, true
}

func hasSupportedImageExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".webp":
		return true
	default:
		return false
	}
}

func containsSensitiveText(value string) bool {
	return credentialPattern.MatchString(value) || openAIKeyPattern.MatchString(value) || privateIPPattern.MatchString(value)
}

func filterEnvKeys(env []string, keys ...string) []string {
	blocked := map[string]bool{}
	for _, key := range keys {
		blocked[key] = true
	}
	filtered := []string{}
	for _, item := range env {
		key, _, ok := strings.Cut(item, "=")
		if ok && blocked[key] {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func finishRenderExecution(rootDir string, execution RenderExecution) (RenderExecution, error) {
	execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := fsutil.WriteJSON(filepath.Join(renderExecutionsDir(rootDir), execution.ID+".json"), execution); err != nil {
		return RenderExecution{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(renderExecutionsDir(rootDir), "events.jsonl"), execution); err != nil {
		return RenderExecution{}, err
	}
	_ = logging.Log(rootDir, "model", "visual.render.execution.created", map[string]any{
		"execution_id": execution.ID,
		"asset_id":     execution.AssetID,
		"decision":     execution.Decision,
		"status":       execution.Status,
		"mode":         execution.Mode,
	})
	return execution, nil
}

func visualsDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).MoyuanDir, "visuals")
}

func assetsDir(rootDir string) string {
	return filepath.Join(visualsDir(rootDir), "assets")
}

func renderExecutionsDir(rootDir string) string {
	return filepath.Join(visualsDir(rootDir), "executions")
}

func previewsDir(rootDir string) string {
	return filepath.Join(visualsDir(rootDir), "previews")
}

func normalizeDiagramType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	if value == "" {
		value = "architecture"
	}
	switch value {
	case "architecture", "issue_graph", "multi_agent", "deployment_topology", "release_flow":
		return value
	default:
		return ""
	}
}

func normalizeRenderMode(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "script", "local_script":
		return "script"
	default:
		return value
	}
}

func scriptFor(diagramType string) string {
	if normalizeDiagramType(diagramType) == "multi_agent" {
		return "scripts/generate-multi-agent-flow-image.js"
	}
	return "scripts/generate-architecture-image.js"
}

func renderCommand(scriptPath string) string {
	return "node " + scriptPath
}

func outputPathFromScript(output string, prefix string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func sanitizeExecutionText(value string) string {
	value = credentialPattern.ReplaceAllString(value, "$1=[REDACTED]")
	value = openAIKeyPattern.ReplaceAllString(value, "[REDACTED_API_KEY]")
	return strings.TrimSpace(value)
}

func defaultTitle(diagramType string) string {
	switch diagramType {
	case "issue_graph":
		return "Moyuan Issue Graph Diagram"
	case "multi_agent":
		return "Moyuan Multi-Agent Orchestration Diagram"
	case "deployment_topology":
		return "Moyuan Deployment Topology Diagram"
	case "release_flow":
		return "Moyuan Release Flow Diagram"
	default:
		return "Moyuan Architecture Diagram"
	}
}

func (options DiagramOptions) TitleOr(fallback string) string {
	if strings.TrimSpace(options.Title) == "" {
		return fallback
	}
	return options.Title
}

func sanitize(value string) string {
	value = credentialPattern.ReplaceAllString(value, "$1=[REDACTED]")
	value = openAIKeyPattern.ReplaceAllString(value, "sk-[REDACTED]")
	value = privateIPPattern.ReplaceAllString(value, "[REDACTED_PRIVATE_IP]")
	return strings.TrimSpace(value)
}

func nodesFor(diagramType string) []DiagramNode {
	switch diagramType {
	case "multi_agent":
		return []DiagramNode{
			node("requirement", "Requirement Refiner", "process", "clarification", "scope", "acceptance"),
			node("issue_graph", "Issue Graph", "state", "DAG", "dependencies", "write scopes"),
			node("scheduler", "Scheduler", "process", "ready/blocked/running/review", "parallelism", "runtime slots"),
			node("subagent", "Subagent Plan", "agent", "role", "skills", "memory scope", "output contract"),
			node("runtime", "Runtime Adapter", "runtime", "Claude CLI", "Codex CLI", "provider route"),
			node("quality", "Quality Gate", "gate", "build", "lint", "test", "review"),
			node("learning", "Learning Loop", "feedback", "memory", "skill effectiveness", "compact"),
		}
	case "deployment_topology":
		return []DiagramNode{
			node("release", "Release Plan", "process", "branch", "tag", "GitHub/Gitee"),
			node("resources", "Server Resources", "state", "test_dev", "production", "expires_at"),
			node("deployment", "Deployment", "process", "dry-run", "controlled execute", "audit"),
			node("smoke", "Smoke Test", "gate", "health", "endpoint", "rollback trigger"),
			node("monitor", "Monitor", "feedback", "logs", "metrics", "incidents"),
		}
	default:
		return []DiagramNode{
			node("repository", "Repository", "source", "local path", "GitHub/Gitee", ".moyuan"),
			node("comprehension", "Project Comprehension", "process", "profile", "module map", "commands"),
			node("planning", "Requirement Planning", "process", "clarification", "Issue Graph", "schedule"),
			node("agents", "Multi-Agent Execution", "agent", "Subagent", "Skills Registry", "Model Routing"),
			node("quality", "Quality & Review", "gate", "diff", "tests", "review"),
			node("memory", "Agent Memory", "state", "record gate", "retrieve", "compact"),
			node("release", "Release & Deploy", "process", "branch", "push", "smoke", "monitor"),
		}
	}
}

func edgesFor(diagramType string) []DiagramEdge {
	nodes := nodesFor(diagramType)
	edges := []DiagramEdge{}
	for idx := 0; idx < len(nodes)-1; idx++ {
		edges = append(edges, DiagramEdge{From: nodes[idx].ID, To: nodes[idx+1].ID, Label: "main_flow"})
	}
	return edges
}

func node(id string, label string, kind string, details ...string) DiagramNode {
	return DiagramNode{ID: id, Label: label, Kind: kind, Details: details}
}

func buildPrompt(spec DiagramSpec) string {
	lines := []string{
		"# Diagram Prompt",
		"",
		"请生成一张横版 2K 技术流程图。",
		"",
		"- Title: " + spec.Title,
		"- Diagram Type: " + spec.DiagramType,
		"- Scope: " + spec.Scope,
		"- Style: white background, dark blue section headers, light cards, clear arrows, technical but readable.",
		"- Keep Chinese explanations concise; preserve English technical terms such as Issue Graph, Subagent, Scheduler, Runtime Adapter, Quality Gate, Agent Memory.",
		"- Do not include API keys, tokens, private IPs, passwords, account names, or raw environment values.",
		"",
		"## Nodes",
	}
	for _, item := range spec.Nodes {
		lines = append(lines, "- "+item.ID+": "+item.Label+" ("+item.Kind+") - "+strings.Join(item.Details, ", "))
	}
	lines = append(lines, "", "## Edges")
	for _, edge := range spec.Edges {
		lines = append(lines, "- "+edge.From+" -> "+edge.To+" ["+edge.Label+"]")
	}
	return strings.Join(lines, "\n") + "\n"
}
