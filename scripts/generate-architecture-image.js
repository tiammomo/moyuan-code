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
    "agent-roles-overview.md",
    "subagents-skills-system.md",
    "agent-memory-system.md",
    "model-tool-adapters.md",
    "configuration-guide.md",
    "configuration-schema-spec.md",
    "repository-onboarding-git-management.md",
    "code-lifecycle-quality-gates.md",
    "engineering-process-standards.md",
    "mainlines/project-comprehension.md",
    "mainlines/requirement-planning.md",
    "mainlines/code-development.md",
    "mainlines/code-management.md",
    "mainlines/runtime-feedback-self-repair.md",
    "mainlines/server-resource-management.md",
    "mainlines/devops-release-deployment.md",
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
图名：Moyuan Code 多 Agent 研发全生命周期技术流程图
目标：展示当前项目从用户需求、仓库接入、项目理解、Issue Graph、Subagent 并发执行、质量复核、Git 合入、发布投产到 Memory/日志反馈的完整技术流程。
受众：技术负责人、架构师、后端/前端/测试/运维 Agent 配置人员、后续实现工程师。

画面布局：
请用横向泳道图或分层流程图。必须包含 7 条技术主线，并用箭头展示主流程和反馈回路。

1. 入口与控制面
   - Platform User / CLI / API / Web Console
   - Auth Context / RBAC / Approval / Audit
   - Secret refs, no plaintext secrets

2. 仓库接入与项目理解
   - Local Path / GitHub / Gitee / Generic Git
   - Git Adapter: clone, fetch, branch, worktree, PR/MR capability
   - .moyuan Workspace initialization
   - Full / Incremental / Diff Project Comprehension
   - Project Profile, Module Map, Commands, Risk Files, Memory Candidates

3. 需求规划与 Issue Graph
   - Requirement Refiner
   - Clarification Gate: needs_user_input or proceed
   - Issue Planner
   - Dependency Planner
   - Scheduler
   - User-visible Issue Graph
   - ready_queue / blocked_queue / running_queue / review_queue

4. Subagent 执行平面
   - Agent Roles Overview: frontend, backend, backend_tuning, tester, quality_guard, reviewer, release_manager
   - Subagent Manager: parent issue/run, role, skills, memory scope, write scope
   - Skills Registry / find-skills / Skill Binding / Effectiveness
   - Runtime Adapter
   - Claude CLI for frontend and architecture-heavy work
   - Codex CLI for backend, tests, review, quality and repair
   - Model Providers: GPT, Claude, GLM, MiniMax, DeepSeek, DashScope, Third-party API

5. 代码质量与合入
   - issue branch / issue worktree
   - code edits and generated tests
   - build / lint / typecheck / unit tests / integration tests
   - coverage, duplication, complexity, architecture boundary, security
   - independent review
   - accepted -> merge to epic integration branch
   - failed -> needs_rework -> replan

6. 版本发布与服务器 DevOps
   - release suggestion and release batch policy
   - release branch, release notes, tag
   - push to GitHub/Gitee, PR/MR
   - environments: test, staging, production
   - server resources: test_dev, production, cloud metadata, expiration, renewal owner
   - deployment, backup, online smoke tests, monitoring window, rollback

7. 反馈闭环和长期治理
   - Runtime Signals -> Bug Candidate -> Repair Attempt -> Improvement Record
   - Agent Memory: Record Gate, Extraction, Staging Dedup, Async Write, Retrieval, Automatic Compact, Reflection
   - Unified Logs: run, agent, model, git, quality, release, deployment, memory, audit, error
   - Documentation Governance, Config Schema, Contracts, Failure Recovery
   - gpt-image-2 Visuals: diagram spec, prompt, image, explanation

视觉要求：
- 生成清晰的工程技术流程图，不要营销风格，不要卡通，不要 3D，不要抽象插画。
- 使用白底、深灰文字、蓝/绿/橙/紫作为泳道或模块区分色。
- 标签要短、粗、清晰；不要大量小字。
- 用实线箭头表示主流程，用虚线箭头表示反馈闭环和治理回路。
- 画面中必须能看出“需求 -> Issue Graph -> Subagent 并发执行 -> 质量门禁 -> Git 合入 -> 发布部署 -> Memory/Logs/自我修复反馈”的主流程。
- 右下角放一个小型图例：solid arrow = main flow, dashed arrow = feedback.
- 不要出现任何 API Key、token、私网 IP、真实账号或密码。
`;
}

function buildPrompt() {
  const docsContext = collectDocsContext();
  return `
你是资深软件架构图设计师。请根据下面的设计规格，生成一张适合技术评审会议展示的技术流程图。

${buildDiagramSpec()}

当前 docs 目录的文档结构摘要：
${docsContext}

输出要求：
- 只生成一张完整架构流程图。
- 图片中不要出现说明性段落，保留必要短标签即可。
- 所有文字使用中文，英文技术名可以保留，例如 Claude CLI、Codex CLI、Issue Graph、Memory、Logs。
- 图需要让工程师一眼看懂当前 Moyuan Code 项目的端到端执行链路、并发编排、质量控制、发布投产和长期反馈闭环。
`;
}

function writeExplanation(outBase, promptPath) {
  const promptDisplayPath = path.relative(root, promptPath);
  const explanation = `# Moyuan Code 技术流程图讲解

这张图展示当前项目的端到端技术流程：用户通过 CLI/API/Web Console 提交项目接入、开发任务、发布或维护请求，系统先建立 Auth Context 和权限边界，再进入仓库接入、项目理解、需求规划、Issue Graph、Subagent 执行、质量门禁、Git 合入、发布投产和长期反馈闭环。

## 1. 入口与控制面

入口层承接 Platform User、CLI、API 和 Web Console。任何操作进入项目之前，都必须先形成 Auth Context，再经过 RBAC、Approval 和 Audit 判断。高风险动作，例如生产部署、Git push、tag、密钥访问、服务器命令和策略变更，不能绕过审批与审计。

## 2. 仓库接入与项目理解

仓库接入支持 Local Path、GitHub、Gitee 和 Generic Git。Git Adapter 负责 clone、fetch、branch、worktree、push、PR/MR 能力声明和用户改动保护。项目接入后会初始化独立 .moyuan 工作空间，并立即触发 Full Project Comprehension；每次远程同步、rebase、merge 或任务完成后触发 Incremental/Diff Comprehension。

阅读理解产物包括 Project Profile、Module Map、Commands、Risk Files 和 Memory Candidates。这些产物不是完整源码复制，而是后续需求规划、Agent 上下文装配、质量判断和记忆检索的稳定事实来源。

## 3. 需求规划与 Issue Graph

用户提出开发任务后，不直接进入编码。Requirement Refiner 会补齐背景、范围、约束、验收和风险；Clarification Gate 判断是否必须追问用户。信息足够后，Issue Planner 拆分 issues，Dependency Planner 构建 DAG，Scheduler 计算 ready_queue、blocked_queue、running_queue 和 review_queue。

Issue Graph 是系统调度的核心。它控制哪些 issue 可以并发、哪些必须等待契约、后端、前端、Runtime slot、worktree、质量门禁或用户审批。用户可以看到 issue graph、blocked reason 和并发计划。

## 4. Subagent 执行平面

Issue 被调度后，Orchestrator 不直接调用模型，而是创建 Subagent Plan。Subagent 绑定父对象、role、skills、memory scope、read/write scope、Runtime 和输出契约。

默认分工是：前端和复杂 UI/架构任务优先使用 Claude CLI；后端、测试、review、quality_guard、repair 和后端调优优先使用 Codex CLI。GPT、Claude、GLM、MiniMax、DeepSeek、DashScope 和第三方 API 通过 Provider Registry 和 Model Routing 参与规划、审查、摘要、Memory record gate、抽取和降级 fallback。

Skills Registry 和 find-skills 负责推荐能力包。Skill Binding 可按 project、role、issue 或 subagent 绑定，并通过 Skill Effectiveness 记录效果，避免长期使用低质量技能。

## 5. 代码质量与合入

每个 issue 使用独立 issue branch 或 issue worktree。Subagent 完成代码修改后，系统必须执行 build、lint、typecheck、unit tests、integration tests、coverage、重复度、复杂度、架构边界和安全检查。

Quality Gate 和 Reviewer 都通过后，issue 才能 accepted，并合入 epic integration branch。失败时进入 needs_rework 或 replan，不允许把未复核的 AI 代码直接合入主分支。

## 6. 版本发布与服务器 DevOps

当 integration branch 累积足够 accepted issues，Release Manager 根据风险、issue 数量、变更范围、迁移、安全和公共 API 变更生成 release suggestion。发布流程包括 release branch、release note、tag、push 到 GitHub/Gitee、PR/MR、回归测试和审批。

投产阶段读取 environments 和 server resources。服务器资源区分 test_dev 和 production，记录云厂商、规格、到期时间、续费 owner、健康检查、备份和维护策略。生产部署必须执行备份、线上冒烟、监控窗口和回滚判断。

## 7. 反馈闭环和长期治理

运行失败、测试失败、冒烟失败、监控异常、review finding 或用户反馈会进入 Runtime Signals。系统先判断是否为 Bug Candidate，再决定是否自动 Repair Attempt。成功修复会生成 Improvement Record，并可能进入 Memory、Skill 效果反馈或质量策略增强。

Agent Memory 通过 Record Gate、Extraction、Staging Dedup、Async Write、Retrieval、Automatic Compact 和 Reflection 管理长期记忆。统一日志记录 run、agent、model、git、quality、release、deployment、memory、audit 和 error，保证每一次自动化行为可追踪。

## 8. .moyuan 工作空间

每个被管理项目都有独立 .moyuan 工作空间。核心目录包括 project、repository、agents、models、runtimes、skills、memory、logs、comprehension、resources、lifecycle 和 policies。配置 Schema、契约文档、状态机和文档治理共同保证后续实现不会把对象字段、流程规则和策略判断散落到多个不一致的位置。

## 9. gpt-image-2 的角色

gpt-image-2 只用于架构图、流程图、部署拓扑图和讲解资产生成。它接收脱敏后的 diagram spec 和视觉 prompt，不参与代码生成、代码审查、质量合入或发布决策。

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
