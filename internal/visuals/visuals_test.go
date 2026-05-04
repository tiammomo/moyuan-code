package visuals

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/providers"
	"moyuan-code/internal/workspace"
)

func TestRenderAssetScriptUsesProviderAuthAndWritesQualityPreview(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	imageProvider, found, err := providers.Show(root, "gpt_image_2")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("missing gpt_image_2 provider")
	}
	imageProvider.Enabled = true
	imageProvider.AuthRef = "env:IMAGE_TEST_KEY"
	imageProvider.BaseURL = "https://image.example/v1"
	if _, err := providers.Upsert(root, imageProvider); err != nil {
		t.Fatal(err)
	}

	plan, err := GeneratePlan(root, DiagramOptions{DiagramType: "multi_agent"})
	if err != nil {
		t.Fatal(err)
	}
	writeFakeImageScript(t, root)
	t.Setenv("MOYUAN_ALLOW_IMAGE_SCRIPT", "1")
	t.Setenv("IMAGE_TEST_KEY", "render-token")

	execution, err := RenderAsset(context.Background(), root, RenderOptions{
		AssetID:  plan.Asset.ID,
		Mode:     "script",
		Approved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Decision != "VISUAL_RENDER_COMPLETED" || execution.Status != "completed" {
		t.Fatalf("expected completed render execution, got %+v", execution)
	}
	if execution.AuthRef != "env:IMAGE_TEST_KEY" || !hasString(execution.EnvKeys, "OPENAI_API_KEY") || !hasString(execution.EnvKeys, "OPENAI_BASE_URL") {
		t.Fatalf("expected auth ref and injected env key audit, got auth=%q env=%v", execution.AuthRef, execution.EnvKeys)
	}
	if execution.Quality == nil || execution.Quality.Status != "passed" {
		t.Fatalf("expected passed quality result, got %+v", execution.Quality)
	}
	if execution.PreviewIndexPath == "" {
		t.Fatalf("expected preview index path in execution: %+v", execution)
	}
	if strings.Contains(fmt.Sprintf("%+v", execution), "render-token") {
		t.Fatalf("render execution leaked API token")
	}
	preview, err := os.ReadFile(execution.PreviewIndexPath)
	if err != nil {
		t.Fatal(err)
	}
	previewText := string(preview)
	if !strings.Contains(previewText, execution.ID) || !strings.Contains(previewText, `"quality_status":"passed"`) {
		t.Fatalf("preview index missing execution or quality status: %s", previewText)
	}
	if strings.Contains(previewText, "render-token") {
		t.Fatalf("preview index leaked API token")
	}
	asset, found, err := LoadAsset(root, plan.Asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found || asset.Status != "rendered" || asset.ImagePath == "" || asset.ExplanationPath == "" {
		t.Fatalf("expected rendered asset with image/explanation paths, found=%v asset=%+v", found, asset)
	}
}

func writeFakeImageScript(t *testing.T, root string) {
	t.Helper()
	scriptPath := filepath.Join(root, "scripts", "generate-multi-agent-flow-image.js")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatal(err)
	}
	script := `const fs = require("node:fs");
const path = require("node:path");
if (process.env.OPENAI_API_KEY !== "render-token") {
  console.error("bad auth");
  process.exit(2);
}
if (process.env.OPENAI_BASE_URL !== "https://image.example/v1") {
  console.error("bad base url");
  process.exit(3);
}
const root = process.cwd();
const outDir = path.join(root, "docs", "assets");
const promptDir = path.join(root, ".moyuan", "visuals", "prompts");
fs.mkdirSync(outDir, { recursive: true });
fs.mkdirSync(promptDir, { recursive: true });
const imagePath = path.join(outDir, "fake-render.png");
const promptPath = path.join(promptDir, "fake-render.prompt.md");
const explanationPath = path.join(outDir, "fake-render.explanation.md");
fs.writeFileSync(imagePath, "fake png");
fs.writeFileSync(promptPath, "safe prompt");
fs.writeFileSync(explanationPath, "safe explanation");
console.log("Image written: " + imagePath);
console.log("Prompt written: " + promptPath);
console.log("Explanation written: " + explanationPath);
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func hasString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
