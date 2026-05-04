# Phase 3 实现 Issue Graph

状态：ready_for_next_phase
责任角色：orchestrator_owner + config_owner + frontend_owner + adapter_owner + qa_owner
最后更新：2026-05-05

Phase 3 的目标是把 Phase 2 已完成的多模型、Skills、Native Runtime、Subagent 调度和 Visual Diagram 能力继续推进到“配置可执行、操作可见、执行可控”的生产化前置阶段。当前第一批 issue 已完成，验收结论见 [Phase 3 Release Readiness](./phase3-release-readiness.md)。

## 1. Phase 3 目标

- `.moyuan/*.yaml` 不再只是文档样例，而是可读取、可校验、可阻断执行的项目配置。
- Console 从可观测面板升级到可执行工作台，逐步承接审批、重试、预览、diff 和日志查看。
- Provider、Visual、Runtime、Server 和 Release 的执行入口继续保持受控，不允许绕过质量门禁。
- Phase 3 不引入分布式队列和生产多租户集群，先把单机控制面做扎实。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase3-001` | `workspace-yaml-schema-validator` | completed | 读取并校验 `.moyuan/project.yaml`、`repository.yaml`、`policies/access.yaml`，检测 YAML 解析错误、条件必填、必须为空和 state drift | Phase 2 release readiness | `config_owner` | `moyuan workspace validate` 能发现用户编辑的 YAML 配置错误 |
| `phase3-002` | `workspace-schema-coverage-expansion` | completed | 将 validator 扩展到 providers、routing、visuals、runtimes、server、release 和 budget 配置 | `phase3-001` | `config_owner` | 核心配置域有字段级 issue code 和文档映射 |
| `phase3-002a` | `providers-yaml-schema-validator` | completed | 校验 `.moyuan/models/providers.yaml` 的 schema、accounts、providers、auth_ref 引用和明文密钥禁用 | `phase3-001` | `config_owner` | provider 配置错误能在执行前被 workspace validate 阻断 |
| `phase3-002b` | `routing-yaml-schema-validator` | completed | 校验 `.moyuan/models/routing.yaml` 的 policies、primary provider、fallback provider 和明文密钥禁用 | `phase3-002a` | `config_owner` | 模型路由策略缺失 provider 能在执行前被阻断 |
| `phase3-002c` | `visuals-yaml-schema-validator` | completed | 校验 `.moyuan/visuals/architecture-visuals.yaml` 的 provider policy、diagram types、pipeline、gpt-image-2 和 safety | `phase3-002a` | `config_owner` | 图像流水线缺少安全配置或生成策略时被阻断 |
| `phase3-002d` | `agent-runtimes-yaml-schema-validator` | completed | 校验 `.moyuan/runtimes/agent-runtimes.yaml` 的 native runtime、auth、env profile、health、audit 和质量门禁 | `phase3-002b` | `config_owner` | Claude/Codex Runtime 配置错误能在执行前被阻断 |
| `phase3-002e` | `devops-policy-yaml-validator` | completed | 校验 `.moyuan/policies/server-resources.yaml`、`environments.yaml`、`release.yaml`、`budget.yaml` 的生产资源、部署环境、发布门禁和预算约束 | `phase3-002d` | `config_owner` + `devops_owner` | 生产机、生产环境、发布部署和并发预算错误能在执行前被阻断 |
| `phase3-003` | `console-operation-actions` | completed | Console 增加受控操作入口：requirement plan、dry-run、runtime artifact preview、release suggest、deploy dry-run、health scan、visual render dry-run | `phase3-001` | `frontend_owner` | 高风险动作只调用后端受控 API，不在前端绕过门禁 |
| `phase3-003a` | `visual-render-dry-run-console-action` | completed | 在 Visual Assets 面板触发后端 dry-run render，并展示 execution id、decision 和错误 | `phase2-008`,`phase3-001` | `frontend_owner` | Console 可触发受控 dry-run，不调用真实图片 API |
| `phase3-004` | `runtime-log-diff-viewer` | completed | Console 展开 runtime recovery 的 stdout/stderr、diff summary、changed files 和 resume hint | `phase2-007`,`phase2-009` | `frontend_owner` | 运行失败能在 Console 看到证据链 |
| `phase3-005` | `provider-probe-adapters` | completed | Provider refresh 接入可选轻量探测 adapter，仍不保存明文密钥 | `phase2-005`,`phase3-001` | `adapter_owner` | 探测失败可解释，不影响禁用 provider 的安全路由 |
| `phase3-006` | `visual-script-auth-quality` | completed | Visual script mode 接入 auth ref、密钥注入审计、图片产物质量检查和预览索引 | `phase2-008`,`phase3-001` | `visualization_owner` | 图片生成可执行但必须有审批、审计和质量结果 |
| `phase3-007` | `release-deploy-control-actions` | completed | Release、deploy、smoke、monitor 的控制台动作与后端 dry-run/approval 对齐 | Beta deploy 基线, `phase3-003` | `devops_owner` | 发布到 GitHub/Gitee 和服务器部署均有可见流水线状态 |

## 3. 执行顺序

1. 先做 `phase3-001`，让配置文件成为可执行事实源，避免后续操作读取到错误配置。
2. `phase3-002` 扩展 schema 覆盖面，但不阻塞 Console 小步增强。
3. `phase3-003`、`phase3-004` 优先提升用户操作和失败排查效率。
4. `phase3-005`、`phase3-006` 接入更真实的外部能力，但必须保留 dry-run、approval 和审计。
5. `phase3-007` 放在操作工作台稳定后推进。

## 4. 收口规则

- 每个 Phase 3 issue 必须有可运行测试或明确的前端构建验证。
- 配置字段、issue code、状态机或执行入口变更必须回写到主线、策略、契约或配置文档。
- 任何真实外部调用必须默认关闭，并提供 dry-run 或 preview。
- AI 生成代码仍必须通过测试、review 和 git 提交记录后才能进入 main。

## 5. 收口结论

- `phase3-001` 到 `phase3-007` 第一批 issue 已完成。
- `workspace validate` 已覆盖项目、仓库、权限、Provider、Routing、Visual、Runtime、Server Resource、Environment、Release 和 Budget 配置。
- Console 已能触发受控的 visual dry-run、runtime artifact preview、release suggest、deployment dry-run 和 resource health scan。
- Provider probe 和 Visual script mode 已具备受控外呼、密钥引用、脱敏审计和质量检查边界。
- 生产级多 worker、真实 PR/MR 创建、真实生产部署、团队级 RBAC 和 secret manager 留到 Phase 4 后续拆分。
