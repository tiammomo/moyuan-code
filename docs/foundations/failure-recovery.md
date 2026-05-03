# 失败恢复设计

本文定义 Moyuan Code 的失败分类、恢复动作和禁止行为。状态机由 [参考架构](../reference-architecture.md) 和 [Issues 编排与并发调度](../issue-orchestration.md) 维护，本文只描述异常场景如何恢复。

## 设计原则

- 失败必须可追踪：记录 run、issue、agent、runtime、命令、错误摘要和相关 diff。
- 失败不能静默跳过：所有失败都必须进入状态机。
- 自动恢复必须有限：达到重试上限后进入人工介入。
- 恢复不能扩大权限：不能因为失败而绕过鉴权、权限、质量门禁或审批。
- 生产失败优先保护服务稳定性。

## 通用失败记录

每次失败至少记录：

- `failure_id`
- `project_id`
- `epic_id`
- `issue_id`
- `run_id`
- `phase`
- `component`
- `error_type`
- `message`
- `retry_count`
- `next_action`
- `created_at`

落盘位置：

- `.moyuan/logs/errors/`
- `.moyuan/lifecycle/runs/`
- 相关模块目录，例如 `quality/`、`deployments/`、`model-ops/incidents/`

## 通用恢复状态

| 状态 | 含义 |
| --- | --- |
| `retrying` | 系统正在按策略自动重试 |
| `fallback_running` | 已切换备用 Runtime、Provider 或策略 |
| `needs_rework` | 需要回到实现阶段修复 |
| `needs_user_input` | 缺少必要信息 |
| `needs_approval` | 需要审批 |
| `needs_human_intervention` | 自动恢复耗尽 |
| `cancelled` | 用户取消 |
| `failed_final` | 无法恢复，保留失败记录 |

## 用户与鉴权失败

触发条件：

- 缺少身份凭证。
- 用户被禁用。
- 会话过期或被撤销。
- API Token 过期、撤销或 scope 不匹配。
- 项目成员关系不存在。
- 审批缺失、过期或被拒绝。

系统动作：

1. 阻断当前操作，写入 `auth.decision.deny` 或 `auth.decision.require_approval` 审计事件。
2. 如果是会话过期，提示重新登录或重新初始化本地身份。
3. 如果是 Token 问题，提示轮换或重新创建 Token。
4. 如果是成员关系或角色不足，生成权限申请或审批请求。
5. 对已经排队但尚未开始的写入、Git、服务器、发布和部署任务重新做鉴权。

禁止：

- 使用过期会话继续审批。
- 使用被撤销 Token 自动重试。
- 将 `DENY` 降级为只记录警告。
- 为恢复任务临时扩大 actor 权限。

恢复出口：

- 用户重新登录。
- Token 轮换成功。
- 成员关系或角色被授权。
- 审批通过。
- 用户取消操作。

## 项目接入失败

触发条件：

- 本地路径不存在。
- 远程 Git URL 无法访问。
- 鉴权失败。
- 目标目录不是 Git 仓库。

系统动作：

1. 标记 project onboarding 为 failed。
2. 记录 provider、url 类型、错误码和建议动作。
3. 如果是远程仓库，尝试区分网络失败、权限失败和仓库不存在。
4. 不创建不完整 workspace，或标记为 `incomplete`。

禁止：

- 猜测替代仓库地址。
- 保存明文 Git 凭证。
- 在接入失败后继续创建任务。

恢复出口：

- 用户修正路径或 URL。
- 用户补充凭证。
- 用户取消项目接入。

## 项目理解失败

触发条件：

- 依赖文件无法读取。
- 项目过大导致上下文超限。
- 语言/框架识别失败。
- 模型或 Runtime 执行失败。

系统动作：

1. 标记 comprehension 为 failed 或 partial。
2. 保留已成功生成的部分产物。
3. 降级为目录级摘要和关键文件抽样。
4. 记录缺失信息和风险。

禁止：

- 将 partial comprehension 当作完整理解。
- 在风险未知时自动执行高风险 issue。

恢复出口：

- 重跑 full comprehension。
- 用户提供项目说明。
- 使用更强 Runtime 或更小范围增量理解。

## 需求澄清失败

触发条件：

- 用户目标无法拆分。
- 关键验收标准缺失。
- 存在互斥约束。
- 涉及高风险业务但缺少确认。

系统动作：

1. 标记 Epic 为 `needs_user_input`。
2. 生成最少数量的澄清问题。
3. 暂停 Issue Graph 生成。

禁止：

- 猜测用户意图并直接开发。
- 把不确定需求拆成可执行 issue。

恢复出口：

- 用户补充信息。
- 用户明确允许按默认策略执行。

## Issue 拆分失败

触发条件：

- 无法识别开发边界。
- Issue 之间依赖循环。
- 写入范围冲突严重。
- 验收标准不完整。

系统动作：

1. 标记 planning 为 failed。
2. 输出失败原因和可修正项。
3. 尝试生成更粗粒度 issues。
4. 如果出现依赖循环，要求 dependency planner 重建图。

禁止：

- 生成不可验收 issue。
- 忽略循环依赖。

恢复出口：

- 重新规划。
- 用户调整需求范围。

## Runtime 执行失败

触发条件：

- Claude CLI 或 Codex CLI 不可用。
- Runtime 超时。
- 退出码非 0。
- 输出无法解析。
- 声称完成但 diff 为空。
- 修改超出 write scope。

系统动作：

1. 保存 stdout、stderr、退出码和 diff。
2. 标记 Run 为 failed。
3. 若策略允许，fallback 到备用 Runtime。
4. 若修改越权，撤销该 Run 的候选改动并进入人工介入。
5. 将 issue 标记为 `needs_rework` 或 `failed`。

禁止：

- 跳过质量门禁。
- 在输出无法解析时直接标记完成。
- 接受越权写入。

恢复出口：

- Runtime 重试成功。
- 备用 Runtime 成功。
- 人工修正后重新运行。

## 模型 API 失败

触发条件：

- 鉴权失败。
- 限流。
- 超时。
- 服务商不可用。
- 结构化输出不合法。
- 数据策略拒绝。

系统动作：

1. 写入 `.moyuan/model-ops/incidents/`。
2. 增加 provider 失败计数。
3. 对可恢复错误执行有限重试。
4. 对结构化输出尝试 schema repair。
5. 达到阈值后禁用 provider 并走 fallback。

禁止：

- 因 fallback 把高敏数据发送给第三方 API。
- 无限重试。
- 忽略 data policy。

恢复出口：

- 原 provider 恢复健康。
- fallback provider 成功。
- 降级为人工确认。

## Git 冲突

触发条件：

- merge/rebase 冲突。
- worktree 不干净。
- 用户新增未跟踪文件。
- base branch 已过期。

系统动作：

1. 暂停当前 issue。
2. 保存冲突文件列表。
3. 运行增量项目理解。
4. 判断是否可自动 replan。
5. 需要时请求用户确认。

禁止：

- 覆盖用户未提交改动。
- 自动 `reset --hard`。
- 强推默认分支。

恢复出口：

- 自动 replan。
- 用户处理冲突。
- 重新创建 issue branch。

## 质量门禁失败

触发条件：

- 测试、lint、build、typecheck 失败。
- 重复代码超过阈值。
- 复杂度过高。
- 架构边界被破坏。
- 安全风险存在。

系统动作：

1. 写入 Quality Report。
2. 标记 Issue 为 `needs_rework`。
3. 将失败项作为最小修复范围交回 implementer。
4. 达到返工上限后进入 reviewer 或用户决策。

禁止：

- 标记 Issue 为 accepted。
- 跳过失败 gate。
- 以 suppress 代替修复，除非有审批和过期时间。

恢复出口：

- 修复后 gate passed。
- 用户接受风险并记录审批。
- Issue 取消或拆分。

## Review 不通过

触发条件：

- reviewer 发现 bug。
- reviewer 发现验收标准未满足。
- reviewer 发现维护性或架构风险。

系统动作：

1. 记录 review findings。
2. 将 issue 标记为 `needs_rework`。
3. 生成返工项。
4. 限制 implementer 只改返工范围。

禁止：

- 由原 implementer 自审通过。
- 忽略 reviewer blocker。

恢复出口：

- 返工后 review accepted。
- 用户确认风险接受。

## Memory 写入失败

触发条件：

- Record Gate 输出失败。
- 结构化抽取失败。
- 暂存去重失败。
- 长期存储写入失败。
- 向量索引失败。

系统动作：

1. 主任务不因异步写入失败而阻塞。
2. 保留 memory candidates。
3. 重试异步写入。
4. 标记 memory health degraded。

禁止：

- 丢弃候选而不记录。
- 写入低置信或敏感内容到长期 Memory。

恢复出口：

- 重试成功。
- Memory Curator 人工处理。
- 候选过期归档。

## 日志写入失败

触发条件：

- 磁盘空间不足。
- 日志文件不可写。
- JSONL 序列化失败。

系统动作：

1. 尝试写入 fallback error log。
2. 标记 run 为 logging_degraded。
3. 对审计日志失败进入 `needs_human_intervention`。

禁止：

- 在审计日志不可写时继续执行高风险操作。
- 丢弃审批、secret 访问、生产部署审计事件。

恢复出口：

- 清理磁盘后重试。
- 切换日志目录。
- 人工确认并补录审计事件。

## 服务器巡检失败

触发条件：

- SSH 连接失败。
- 磁盘超阈值。
- 服务健康检查失败。
- 云资源即将到期。
- 备份缺失。

系统动作：

1. 写入 `.moyuan/resources/checks/`。
2. 更新资源状态。
3. 对生产机生成 maintenance issue。
4. 到期风险按等级提醒负责人。

禁止：

- 忽略生产机健康失败。
- 自动执行破坏性修复。

恢复出口：

- 运维修复。
- 续费记录完成。
- 资源退役或替换。

## Release 失败

触发条件：

- release branch 创建失败。
- 回归测试失败。
- release note 生成失败。
- tag 或 push 失败。

系统动作：

1. 标记 Release 为 failed。
2. 保留 release branch 和测试结果。
3. 阻止 deployment。
4. 根据失败类型创建修复 issue。

禁止：

- 未通过回归直接部署。
- tag 失败后伪造发布完成。

恢复出口：

- 修复后重新 prepare release。
- 用户取消 release。

## Deployment 失败

触发条件：

- precheck 失败。
- artifact 拉取失败。
- 部署命令失败。
- smoke test 失败。
- monitor window 异常。

系统动作：

1. 标记 Deployment 为 failed 或 rollback_required。
2. 停止后续服务器批次。
3. 执行已配置回滚策略。
4. 记录部署日志、冒烟结果和监控窗口。
5. 生成事故或修复 issue。

禁止：

- 忽略线上冒烟失败。
- 在 rollback plan 缺失时继续生产部署。
- 手动绕过 release/deploy pipeline。

恢复出口：

- 回滚成功。
- 修复后重新部署。
- 人工介入恢复。

## 回滚失败

触发条件：

- rollback command 失败。
- previous release 不存在。
- 数据迁移不可逆。
- 回滚后 smoke test 仍失败。

系统动作：

1. 标记为 `needs_human_intervention`。
2. 停止自动部署。
3. 保留完整日志和当前版本信息。
4. 通知负责人。
5. 生成 incident report。

禁止：

- 继续尝试未知命令。
- 清理关键现场日志。

恢复出口：

- 人工恢复。
- 热修复发布。
- 服务降级或切流。

## gpt-image-2 生成失败

触发条件：

- 图像 API 超时。
- prompt 被拒绝。
- 返回格式不兼容。
- 图片可读性检查失败。
- 图和 diagram spec 不一致。

系统动作：

1. 保存 diagram spec 和 prompt。
2. 记录 image incident。
3. 简化图像 prompt 后重试。
4. 仍失败则保留 Markdown 讲解，不阻塞代码生命周期。

禁止：

- 把敏感信息加入 prompt 重试。
- 将不可读图标记为发布产物。

恢复出口：

- 重试成功。
- 人工编辑 diagram spec。
- 只发布文字讲解。
