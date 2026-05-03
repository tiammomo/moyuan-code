#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");

const root = path.resolve(__dirname, "..");
const docsDir = path.join(root, "docs");
const outputDir = path.resolve(root, process.env.OUTPUT_DIR || "docs/assets");
const promptOutputDir = path.resolve(
  root,
  process.env.PROMPT_OUTPUT_DIR || ".moyuan/visuals/prompts"
);

const baseUrl = (process.env.OPENAI_BASE_URL || "https://api.openai.com/v1").replace(/\/+$/, "");
const apiKey = process.env.OPENAI_API_KEY;
const model = process.env.OPENAI_IMAGE_MODEL || "gpt-image-2";
const imagePath = process.env.IMAGE_API_PATH || "/images/generations";
const size = process.env.IMAGE_SIZE || "3072x2048";
const quality = process.env.IMAGE_QUALITY || "high";
const outputFormat = process.env.IMAGE_OUTPUT_FORMAT || "png";
const timeoutMs = Number(process.env.IMAGE_TIMEOUT_MS || 300000);

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
    "issue-orchestration.md",
    "agent-roles-overview.md",
    "subagents-skills-system.md",
    "agent-memory-system.md",
    "model-tool-adapters.md",
    "code-lifecycle-quality-gates.md",
    "engineering-process-standards.md",
    "contracts/subagent-skill-contract.md",
    "policies/provider-routing-policy.md",
    "policies/quality-gate-policy.md",
    "mainlines/requirement-planning.md",
    "mainlines/code-development.md",
    "mainlines/runtime-feedback-self-repair.md",
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
图名：Moyuan Code Multi-Agent Orchestration Flow
目标：生成一张横版 2K 技术流程图，只聚焦 Moyuan Code 的多 Agent 作业流程，不展示完整 SDLC 总流程。
受众：负责实现 Orchestrator、Scheduler、Runtime Adapter、Subagent、Skills、Quality Gate 的工程师。

整体版式：
- 横版 2K 技术调用逻辑图，白底、浅灰卡片、深蓝标题条。
- 顶部大标题：Moyuan Code Multi-Agent Orchestration Flow
- 右上角图例：实线 = 主执行流，虚线 = 控制/反馈，圆柱 = Workspace State，菱形 = Gate。
- 使用编号模块、箭头、队列、DAG、Subagent 卡片、Runtime 图标、质量门禁和反馈闭环。
- 不要人物肖像，不要宣传海报感，不要无关业务场景。

第一行：从左到右展示 7 个主流程模块。

1. 任务入口与上下文
   - User Request
   - Project Profile
   - Agent Memory Retrieve
   - Requirement Refiner

2. 澄清与拆分
   - Clarification Gate
   - Issue Planner
   - Dependency Planner
   - Issue Graph

3. 调度决策
   - Scheduler
   - ready / blocked / running / review queue
   - parallelism budget
   - write scope conflict check

4. Subagent Plan
   - role resolve
   - skills binding
   - memory scope
   - read/write scope
   - output contract

5. Runtime Dispatch
   - Claude CLI for frontend / design
   - Codex CLI for backend / test / review
   - GPT / Claude / GLM / MiniMax API
   - Runtime Adapter

6. Output Convergence
   - collect outputs
   - validate contract
   - diff snapshot
   - command / test report

7. Quality & Merge Decision
   - Build / Lint / Test
   - Quality Gate
   - Reviewer
   - accepted -> merge
   - failed -> rework

第二行：用 DAG 和队列表达并发编排。
- 左侧画 Issue Graph DAG：contract issue -> backend issue + frontend issue -> integration test issue。
- 中间画四个队列：blocked_queue、ready_queue、running_queue、review_queue。
- 右侧画并发计算公式的简化短句：
  parallelism = min(policy, ready issues, worktrees, runtime slots, budget, no write conflicts)
- 用清晰箭头表示：依赖 accepted 后，下游 issue 解锁。

第三行：展示 Subagent 类型和默认分工，放成 6 个小卡片。
- planning_subagent：需求澄清 / issue 拆分
- discovery_subagent：项目阅读 / 模块定位
- implementation_subagent：frontend / backend / tuning
- verification_subagent：tester / quality_guard / reviewer
- repair_subagent：bug_triager / repair_agent
- memory_subagent：memory_curator / compact

右侧竖向控制面：
8. Skills Registry
   - find-skills
   - compatibility score
   - bind / execute / effectiveness
9. Model Routing
   - role based routing
   - capability / cost / data policy
   - fallback
10. Policy Guard
   - auth_context
   - protected path
   - secret ref
   - audit log

底部反馈闭环：
11. Review Findings
    - duplicate code
    - complexity
    - missing tests
12. Rework Loop
    - needs_rework
    - replan
    - retry subagent
13. Learning Loop
    - memory candidates
    - skill effectiveness
    - automatic compact

视觉要求：
- 普通说明用中文，技术专有名词保留英文：Orchestrator、Scheduler、Issue Graph、Subagent、Skills Registry、find-skills、Runtime Adapter、Claude CLI、Codex CLI、Quality Gate、Reviewer、Agent Memory、Memory Compact。
- 每个模块只放 3-5 个核心点，避免长段落。
- 多用图形表达：DAG、队列、门禁菱形、Subagent 卡片、Runtime 插槽、反馈箭头。
- 画面要比总架构图更聚焦多 Agent 编排细节。
- 不要出现 API Key、token、账号、真实服务器 IP、密码。
`;
}

function buildPrompt() {
  return `
你是资深软件架构图设计师。请根据下面规格生成一张横版 2K 技术流程图，主题只聚焦 Moyuan Code 的多 Agent 作业编排。

${buildDiagramSpec()}

当前相关 docs 结构摘要：
${collectDocsContext()}

输出要求：
- 只生成一张完整横版技术调用逻辑图。
- 普通业务动作和说明尽量使用中文；英文技术专有名词必须原样保留。
- 重点突出 Issue Graph、Scheduler、Subagent Plan、Runtime Dispatch、Output Convergence、Quality Gate、Rework Loop 和 Learning Loop。
- 不要把规格里的长句原样塞进图片，用短标签、箭头、图标和模块关系表达。
`;
}

function writeExplanation(outBase, promptPath) {
  const promptDisplayPath = path.relative(root, promptPath);
  const explanation = `# Moyuan Code Multi-Agent 流程图讲解

这张图只聚焦 Moyuan Code 的多 Agent 作业链路：用户需求不会直接进入编码，而是先经过需求完善、澄清判断、Issue Graph 拆分、Scheduler 并发计算，再生成 Subagent Plan 并路由到 Claude CLI、Codex CLI 或模型 API。

## 1. 从需求到 Issue Graph

入口层先装配 Project Profile 和 Agent Memory Retrieve，再由 Requirement Refiner 补齐背景、范围、验收和风险。Clarification Gate 负责判断是否必须追问用户。信息足够后，Issue Planner 拆分 issues，Dependency Planner 生成 Issue Graph。

Issue Graph 是多 Agent 并发的前提。它表达前置依赖、契约依赖、测试依赖、资源冲突和 review 依赖。下游 issue 必须等待上游 accepted 后才能解锁。

## 2. Scheduler 如何决定并发

Scheduler 不使用固定并发数，而是综合 project policy、ready issue 数量、worktree、runtime slot、模型预算和写入范围冲突动态计算。存在同文件写入、核心模块冲突、鉴权/安全/数据库迁移等高风险情况时，自动串行化。

系统维护 blocked_queue、ready_queue、running_queue 和 review_queue。每个 blocked issue 都必须给出 blocked reason，例如 waiting_contract、waiting_runtime_slot、waiting_worktree、waiting_quality 或 waiting_user_input。

## 3. Subagent Plan

Issue 进入 ready 状态后，Orchestrator 创建 Subagent Plan，而不是直接把任务交给模型。Subagent Plan 会声明 role、skills、memory scope、read/write scope、runtime、输出契约和完成条件。

Subagent 是一次受控执行实例，不是长期角色，也不是模型本身。它必须挂在 Epic、Issue、Run、Repair Attempt、Release、Deployment 或 Memory Maintenance Job 等父对象下。

## 4. Runtime Dispatch

Runtime Adapter 负责把 Subagent Plan 转换成具体执行。默认策略是：前端、UI 和复杂设计优先 Claude CLI；后端、测试、review、quality_guard、repair 和后端调优优先 Codex CLI；GPT、Claude、GLM、MiniMax 等 API 用于规划、总结、分类、审查辅助和 memory 抽取。

所有 Runtime 输出都必须回到 Orchestrator，不允许直接合入分支。

## 5. 输出收敛与质量门禁

Orchestrator 收集 Subagent 输出后，先校验输出契约，再读取 diff snapshot、命令记录和测试报告。之后进入 Build、Lint、Test、Quality Gate 和 Reviewer。

只有 Quality Gate 与 Review 都通过，issue 才能 accepted 并合入 integration branch。失败时进入 needs_rework、replan 或 retry subagent，不允许把未复核的 AI 代码直接合入。

## 6. Skills 与学习闭环

Skills Registry 通过 find-skills、技术栈匹配、role 匹配、风险等级、历史成功率和质量效果给 Subagent 绑定 skills。执行完成后记录 skill effectiveness。

Review findings、失败原因、修复经验、测试缺口和有效 skills 会进入 Learning Loop。Memory Curator 负责 memory candidates、去重、冲突处理和 automatic compact，让系统越使用越能减少重复错误和无效返工。

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
  const assetName = `moyuan-code-multi-agent-flow-${timestamp}`;
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
