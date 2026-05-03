#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");

const root = path.resolve(__dirname, "..");
const docsDir = path.join(root, "docs");
const outputDir = path.resolve(
  root,
  process.env.OUTPUT_DIR || "docs/assets"
);
const promptOutputDir = path.resolve(
  root,
  process.env.PROMPT_OUTPUT_DIR || ".moyuan/visuals/prompts"
);

const baseUrl = (process.env.OPENAI_BASE_URL || "https://api.openai.com/v1").replace(/\/+$/, "");
const apiKey = process.env.OPENAI_API_KEY;
const model = process.env.OPENAI_IMAGE_MODEL || "gpt-image-2";
const imagePath = process.env.IMAGE_API_PATH || "/images/generations";
const size = process.env.IMAGE_SIZE || "1536x1024";
const quality = process.env.IMAGE_QUALITY || "high";
const outputFormat = process.env.IMAGE_OUTPUT_FORMAT || "png";
const timeoutMs = Number(process.env.IMAGE_TIMEOUT_MS || 180000);

if (!apiKey) {
  console.error("Missing OPENAI_API_KEY.");
  process.exit(1);
}

function readIfExists(file) {
  if (!fs.existsSync(file)) return "";
  return fs.readFileSync(file, "utf8");
}

function extractHeadings(markdown) {
  return markdown
    .split(/\r?\n/)
    .filter((line) => /^#{1,4}\s+/.test(line))
    .slice(0, 60)
    .join("\n");
}

function collectDocsContext() {
  const files = [
    "README.md",
    "reference-architecture.md",
    "lifecycle-roadmap.md",
    "project-workspace-spec.md",
    "issue-orchestration.md",
    "agent-skills-memory.md",
    "agent-memory-system.md",
    "model-tool-adapters.md",
    "configuration-guide.md",
    "repository-onboarding-git-management.md",
    "code-lifecycle-quality-gates.md",
  ];

  return files
    .map((name) => {
      const fullPath = path.join(docsDir, name);
      const content = readIfExists(fullPath);
      if (!content) return null;
      return `## ${name}\n${extractHeadings(content)}`;
    })
    .filter(Boolean)
    .join("\n\n");
}

function buildDiagramSpec() {
  return `
图名：Moyuan Code 多 Agent 代码开发框架总体结构设计图
目标：展示当前项目规划中的核心模块、执行流程、外部系统和反馈闭环。
受众：技术负责人、架构师、后端/前端/测试/运维 Agent 配置人员。

画面布局：
1. 左侧：用户入口和项目接入
   - User
   - CLI / API / Web Console
   - 本地仓库 / GitHub / Gitee
   - 项目理解 Project Comprehension

2. 中央顶部：Orchestrator 编排核心
   - Requirement Refiner
   - Clarification Gate
   - Issue Planner
   - Dependency Planner
   - Scheduler
   - Issue Graph / Ready Queue

3. 中央中部：Agent Runtime 和执行后端
   - Native Agent Runtime: Claude CLI, Codex CLI
   - Model API Providers: GPT, Claude, GLM, MiniMax, DeepSeek, DashScope, Third-party API
   - Agent Roles: planner, architect, backend, frontend, tester, reviewer, quality_guard, release_manager
   - Skills Engine / find-skills

4. 中央底部：项目工作空间 .moyuan
   - project.yaml / repository.yaml
   - agents / models / runtimes / visuals
   - memory / logs / resources / lifecycle
   - policies: permissions, quality, orchestration, release, environments

5. 右侧：代码生命周期流水线
   - issue worktree / task branch
   - code generation and edits
   - tests / lint / build
   - quality gates
   - review
   - merge into epic integration branch
   - release branch
   - publish to GitHub/Gitee
   - deploy to test_dev / production resource groups
   - online smoke / monitor / rollback

6. 底部反馈闭环：
   - Agent Memory: record gate, extraction, staging dedup, compact, retrieval
   - Unified Logs: run, agent, model, git, quality, release, memory, audit, error
   - Server Resources: cloud metadata, expiration, renewal, checks, maintenance
   - gpt-image-2 Visuals: architecture diagrams, flow explanations

视觉要求：
- 生成清晰的工程架构图，不要营销风格，不要卡通，不要 3D。
- 使用分层架构图 + 流程箭头，节点少而清楚。
- 中文标签要大、短、清晰，避免小字密集。
- 使用冷静专业配色：白底、深灰文字、蓝/绿/橙作为模块区分色。
- 画面中要能看出“需求 -> Issue Graph -> 多 Agent 并发执行 -> 质量门禁 -> 发布部署 -> Memory/Logs 反馈”的主流程。
- 不要出现任何 API Key、token、私网 IP、真实账号或密码。
`;
}

function buildPrompt() {
  const docsContext = collectDocsContext();
  return `
你是资深软件架构图设计师。请根据下面的设计规格，生成一张适合技术评审会议展示的架构流程设计图。

${buildDiagramSpec()}

当前 docs 目录的文档结构摘要：
${docsContext}

输出要求：
- 只生成一张完整架构流程图。
- 图片中不要出现说明性段落，保留必要短标签即可。
- 所有文字使用中文，英文技术名可以保留，例如 Claude CLI、Codex CLI、Issue Graph、Memory、Logs。
- 架构图需要让人一眼看懂当前 Moyuan Code 项目总体设计。
`;
}

function writeExplanation(outBase, promptPath) {
  const promptDisplayPath = path.relative(root, promptPath);
  const explanation = `# Moyuan Code 总体结构设计图讲解

这张图展示当前项目的规划结构：用户通过 CLI/API/Web Console 接入本地或远程仓库，系统先进行项目理解，再由 Orchestrator 完成需求完善、澄清判断、Issue 拆分、依赖图构建和并发调度。

执行层由 Native Agent Runtime 和模型 API 共同组成。Claude CLI 与 Codex CLI 作为强 Agent 后端，负责复杂代码任务；GPT、Claude、GLM、MiniMax、DeepSeek、DashScope 和第三方 API 通过 Provider Registry 统一管理，并按 routing 策略参与规划、审查、摘要和记忆抽取。

每个项目拥有独立的 .moyuan 工作空间，保存项目配置、仓库策略、Agent 角色、模型配置、Runtime 会话、Memory、Logs、服务器资源、生命周期记录和质量策略。

代码生命周期从 Issue Graph 进入独立 worktree 或任务分支，经过代码生成、测试、lint、build、质量门禁、独立 review 后，才能合入 epic integration branch。发布阶段创建 release branch，推送 GitHub/Gitee，并按服务器资源组部署到测试开发机或生产机，随后执行线上冒烟、监控和回滚判断。

底部反馈闭环包括 Agent Memory、统一日志、服务器资源长期维护和 gpt-image-2 架构可视化。它们共同保证系统能持续理解项目、追踪决策、控制质量并辅助讲解架构。

生成文件：

- 图片：${path.basename(outBase)}.${outputFormat}
- Prompt：${promptDisplayPath}
- 讲解：${path.basename(outBase)}.explanation.md
`;
  fs.writeFileSync(`${outBase}.explanation.md`, explanation, "utf8");
}

async function generateImage(prompt) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(`${baseUrl}${imagePath}`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${apiKey}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        model,
        prompt,
        n: 1,
        size,
        quality,
        output_format: outputFormat,
        background: "opaque",
        moderation: "auto",
      }),
      signal: controller.signal,
    });

    const text = await response.text();
    let json;
    try {
      json = JSON.parse(text);
    } catch {
      throw new Error(`Image API returned non-JSON response: ${text.slice(0, 500)}`);
    }

    if (!response.ok) {
      const message = json.error?.message || text;
      throw new Error(`Image API failed (${response.status}): ${message}`);
    }

    const item = json.data?.[0];
    if (!item) {
      throw new Error("Image API response has no data[0].");
    }

    if (item.b64_json) {
      return Buffer.from(item.b64_json.replace(/^data:image\/\w+;base64,/, ""), "base64");
    }

    if (item.url) {
      const imageResponse = await fetch(item.url);
      if (!imageResponse.ok) {
        throw new Error(`Failed to download image URL (${imageResponse.status}).`);
      }
      return Buffer.from(await imageResponse.arrayBuffer());
    }

    throw new Error("Image API response has neither b64_json nor url.");
  } finally {
    clearTimeout(timer);
  }
}

async function main() {
  fs.mkdirSync(outputDir, { recursive: true });
  fs.mkdirSync(promptOutputDir, { recursive: true });
  const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
  const assetName = `moyuan-code-architecture-${timestamp}`;
  const outBase = path.join(outputDir, assetName);
  const promptPath = path.join(promptOutputDir, `${assetName}.prompt.md`);
  const prompt = buildPrompt();

  fs.writeFileSync(promptPath, prompt, "utf8");
  const image = await generateImage(prompt);
  fs.writeFileSync(`${outBase}.${outputFormat}`, image);
  writeExplanation(outBase, promptPath);

  console.log(`Image written: ${outBase}.${outputFormat}`);
  console.log(`Prompt written: ${promptPath}`);
  console.log(`Explanation written: ${outBase}.explanation.md`);
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
