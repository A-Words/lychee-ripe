# AGENTS.md

## 1. 文档定位
- 本文档默认面向执行任务的自动化 Agent，提供可执行的项目约束与工作流程。
- 目标是降低协作分歧：先对齐事实，再执行改动，再完成校验。
- 本文档不替代接口契约与代码实现：
  - 接口与字段以 `shared/schemas/openapi.yaml` 为准。
  - 成熟度映射以 `shared/constants/ripeness.json` 为准。
  - 产品需求与范围以 `docs/prd.md` 为准。

## 2. 项目快照
- 项目目标：荔枝目标检测与成熟度识别，提供后端推理、网关治理与前端可视化。
- 调用链路：`frontend -> gateway -> app`（前端默认只经网关访问后端）。
- 网关实现：Go 网关当前为 HTTP 反向代理 + WebSocket 透传（非 gRPC 调用）。
- 成熟度类别映射（4 类）：`0=green`，`1=half`，`2=red`，`3=young`。
- 关键目录：
  - `app/`：FastAPI 推理服务
  - `gateway/`：Go 网关（鉴权、限流、日志、代理）
  - `frontend/`：Nuxt Web + Tauri Desktop
  - `training/`：训练与评估脚本
  - `tests/`：单元/集成/性能测试
  - `shared/`：常量与契约
  - `configs/`：配置模板与本地配置
  - `scripts/`：脚本入口
  - `artifacts/`：模型与指标产物
- 关键路径：
  - 训练输出：`artifacts/models/`
  - 评估输出：`artifacts/metrics/`
  - 推理模型配置：`configs/model.yaml`
  - 网关配置：`configs/gateway.yaml`
  - 对外契约：`shared/schemas/openapi.yaml`
  - PRD 文档：`docs/prd.md`

## 3. 执行工作流
### 3.1 依赖与配置准备
- 安装依赖：
  - `uv sync`
  - `bun install --cwd frontend`
- 准备本地配置（从 `.example` 复制）：
  - `configs/model.yaml.example -> configs/model.yaml`
  - `configs/service.yaml.example -> configs/service.yaml`
  - `configs/gateway.yaml.example -> configs/gateway.yaml`
- 依赖锁切换（可选，建议用脚本）：
  - `sh scripts/switch-lock.sh --target cpu|cu128|auto`
  - `powershell -ExecutionPolicy Bypass -File scripts/switch-lock.ps1 -Target cpu|cu128|auto`

### 3.2 服务启动
- Agent 默认使用分服务直接命令启动（更利于隔离问题、单服务重启与日志定位）：
  - app（FastAPI）：
    - `uv run uvicorn app.main:app --reload --host 127.0.0.1 --port 8000`
  - gateway（Go）：
    - `go run ./gateway/cmd/gateway --config configs/gateway.yaml`
  - frontend（Web）：
    - `bun run --cwd frontend dev -- --host 127.0.0.1 --port 3000`
  - frontend（Desktop）：
    - `bun run --cwd frontend tauri:dev`

### 3.3 训练与评估
- 训练：
  - `uv run python training/train.py --data data/lichi/data.yaml --model yolo26n.pt --project artifacts/models --name lychee_v1`
- 评估：
  - `uv run python training/eval.py --model artifacts/models/lychee_v1/weights/best.pt --data data/lichi/data.yaml --output artifacts/metrics/lychee_v1-eval_metrics.json`

### 3.4 校验与测试
- Python 测试：`uv run pytest -q`
- Go 测试：`go test ./gateway/...`
- 前端：
  - `bun run --cwd frontend typecheck`
  - `bun run --cwd frontend test`
  - `bun run --cwd frontend generate`

### 3.5 脚本入口（可选）
- 若需标准化复现或减少命令输入，可使用 `scripts/` 下对应封装：
  - 启动：`app.*`、`gateway.*`、`frontend.*`、`desktop.*`、`stack.*`
  - 训练/评估：`train.*`、`eval.*`
  - 校验：`verify.*`
  - 锁切换：`switch-lock.*`

## 4. 改动规则
- 优先最小改动：只改与当前任务直接相关文件。
- 不重命名顶层目录：`app/`、`gateway/`、`training/`、`tests/`、`frontend/`、`shared/`、`configs/`、`scripts/`。
- 改配置或路径时，必须同步检查：
  - `README.md` 示例命令
  - `configs/*.yaml.example`
  - 相关测试与命令参数（若使用脚本，还需检查脚本参数）
- 当改动会影响使用方式、命令、配置、行为或接口契约时，Agent 需同步更新相关文档（至少 `README.md`、`AGENTS.md`、`configs/*.yaml.example`，必要时补充 `docs/`）。
- 改 Python 依赖时，需同步更新 `uv.lock.cpu` 与 `uv.lock.cu128`；提交前恢复 `uv.lock` 为 CPU 基线（与 `uv.lock.cpu` 一致）。
- 新增共享字段时，优先在 `shared/` 定义并同步 `app/`、`gateway/`、`frontend/`。
- 前端禁止直连 `app/`，默认必须走 `gateway/`。
- 任何行为改动，至少运行一次相关测试；若未执行，需明确说明原因与风险。

## 5. 提交前检查
- 命令与路径示例可运行，且参数与脚本实现一致。
- 未引入硬编码绝对路径。
- 未破坏 `configs/*.yaml.example` 可用性。
- 类别映射保持一致：`green/half/red/young`。
- 接口契约一致性：
  - `shared/schemas/openapi.yaml` 与 `app/`、`gateway/`、`frontend/` 字段一致。
- 前端质量门禁通过：`typecheck`、`test`、`generate`。
- 调用链未偏离：`frontend -> gateway -> app`。
- 前端颜色与标签映射与 `shared/constants/ripeness.json` 一致。
- 摄像头切换能力可用（空闲/识别中切换、拔插后刷新、上次选择恢复）。
- 测试执行情况已记录（通过或未执行原因）。

## 6. 默认策略
- 不确定时选择兼容方案：优先保证现有 API、训练流程与脚本入口稳定。
- 涉及结构性改动时，先给迁移步骤与影响范围，再实施变更。
