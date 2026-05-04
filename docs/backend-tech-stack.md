# 后端技术栈与本地环境

状态：ready
责任角色：architect + core_engineer
最后更新：2026-05-04

本文只回答三件事：

1. Moyuan 的后端为什么采用 `Go + Python`。
2. 本地 `Go` 环境和 `Python` 环境分别怎么使用。
3. 两个环境在项目里如何分工、协作和落盘。

它不重复模型适配、工作空间 schema、策略树和模块边界的完整定义。

## 1. 结论

- 控制面后端使用 `Go`。
- 模型邻接、文本处理和轻量执行辅助使用 `Python`。
- `Go` 是唯一的状态与编排主入口。
- `Python` 只做 worker / helper，不直接拥有权威状态。
- 两者之间优先使用版本化 JSON 协议；规模上来后再升级到 `gRPC`。

如果仓库里还保留过渡性的 `Node` 原型，只把它视为临时验证层，不作为最终后端实现。

## 2. 为什么这样分工

### `Go` 负责控制面

适合承担这些职责：

- CLI、API server、鉴权、审计。
- 项目接入、仓库管理、分支管理、worktree 管理。
- issue graph、调度器、并发控制、任务状态机。
- workspace 读写、锁、迁移、原子落盘。
- 质量门禁、review 门禁、发布编排、服务器资源管理。

`Go` 的优势是静态二进制、并发模型清晰、系统集成强、长期运行稳定。

### `Python` 负责模型邻接能力

适合承担这些职责：

- prompt 组装和清洗。
- Memory 抽取、压缩、分类和归档辅助。
- 文本分析、规则化后处理、轻量评审辅助。
- 低风险批处理任务和模型侧工具脚本。
- 必要时再扩展成独立 worker 服务。

`Python` 的优势是模型生态、文本处理生态和快速迭代效率。

## 3. 推荐 Go 技术栈

建议的基础组合：

- HTTP 层：`net/http` + `chi`
- CLI 层：`cobra`
- 配置与序列化：`yaml.v3`、`encoding/json`
- 日志：`slog` 或 `zap`
- 测试：标准库 `testing`、`httptest`
- 进程执行：标准库 `os/exec`

原则：

- 业务入口统一落在 `cmd/moyuan`。
- 领域代码放在 `internal/`。
- 不把模型调用逻辑直接塞进控制面业务函数里。

## 4. 推荐 Python 技术栈

建议的基础组合：

- 包管理：优先 `uv`，否则 `venv + pip`
- 配置：`pyproject.toml`
- 数据校验：`pydantic`
- 测试：`pytest`
- 格式化与检查：`ruff`

可选扩展：

- 如果后续需要 Python 常驻服务，再引入 `FastAPI + Uvicorn`。
- 如果只做 worker，则保留命令行式运行，不必先服务化。

原则：

- Python 只消费 Go 传入的任务规范和上下文摘要。
- Python 不直接修改 Git、workspace 和权威状态存储。

## 5. 本地 Go 环境怎么用

### 安装与检查

```bash
go version
go env GOPATH GOMODCACHE GOROOT
```

建议使用项目约定的稳定版本，通常保持 `Go 1.22+`。

### 常用命令

```bash
go mod download
go test ./...
go run ./cmd/moyuan --help
./bin/moyuan --help
gofmt -w .
```

### 开发约定

- 代码格式由 `gofmt` 统一。
- 单元测试优先覆盖控制面、调度器和状态机。
- 本地构建尽量保持单二进制输出，便于 CLI 分发。
- 交叉编译和发布阶段再处理 `CGO_ENABLED=0`、平台差异和打包细节。

## 6. 本地 Python 环境怎么用

### 推荐方式：`uv`

```bash
uv sync
uv run pytest
uv run python -m moyuan_worker
```

### 兼容方式：`venv + pip`

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -U pip
pip install -e .[dev]
pytest
```

### 开发约定

- Python 代码格式化和检查交给 `ruff`。
- 任务接口、输入输出 schema 和错误类型都要显式化。
- worker 进程应尽量无状态，方便重试和并发。

## 7. Go 与 Python 如何协作

默认协作方式：

1. `Go` 生成结构化任务 spec。
2. `Go` 把脱敏后的上下文传给 `Python` worker。
3. `Python` 返回结构化结果、摘要、候选项或建议。
4. `Go` 负责落盘、审批、调度、审计和后续状态流转。

推荐的传输顺序：

- `MVP`：`stdin/stdout JSON`
- `Beta`：本地 `gRPC`
- `Production`：`gRPC` 或受控 worker pool

约束：

- 任何会影响 Git、发布、审批和权限的动作，都必须由 `Go` 控制面决定。
- `Python` 的输出只能作为建议、候选结果或可验证产物，不能绕过门禁直接生效。

## 8. 推荐目录形态

```text
cmd/
  moyuan/
internal/
  cli/
  api/
  auth/
  orchestrator/
  scheduler/
  workspace/
  git/
  logging/
  quality/
  memory/
  comprehension/
  release/
  serverresources/
  providers/
workers/
  python/
    src/
      moyuan_worker/
scripts/
```

说明：

- `cmd/moyuan` 只放入口。
- `internal` 放 Go 控制面领域代码。
- `workers/python` 放模型邻接任务和辅助脚本。
- `scripts` 只放明确的工具脚本，不承载核心后端逻辑。

## 9. 与其他文档的关系

- 语言与职责决策记录见 [ADR-0005](./adr/0005-go-control-plane-python-worker.md)。
- 模块边界见 [实现模块拆分](./implementation-module-map.md)。
- 模型、Claude CLI、Codex CLI 和 gpt-image-2 的适配边界见 [模型与工具适配规划](./model-tool-adapters.md)。
- 配置入口与环境切分见 [配置方案](./configuration-guide.md)。
