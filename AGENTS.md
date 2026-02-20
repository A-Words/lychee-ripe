# AGENTS.md

## 1. 项目目标
- 项目名称：`lychee-ripe`
- 目标：荔枝目标检测与成熟度识别，并提供后端推理服务与前端可视化能力。
- 主要类别映射（4 类）：`0=green`，`1=half`，`2=red`，`3=young`。

## 2. 目录约定
- `app/`：FastAPI 推理服务与 API（生产推理主路径）。
- `gateway/`：Go 网关服务（对外 API、鉴权、限流、编排、观测），通过 HTTP/gRPC 调用 `app/`。
- `training/`：训练与评估脚本。
- `tests/`：单元/集成/性能测试。
- `frontend/`：前端可视化客户端。
- `shared/`：前后端共享常量与协议（如 `shared/constants/ripeness.json`）。
- `configs/`：配置模板与本地配置（`.example` 可提交，本地 `.yaml` 不提交）。
- `data/`：数据工作区（原始、处理后、样例）。
- `artifacts/`：训练产物、评估指标、日志（模型统一放这里）。
- `scripts/`：自动化脚本（训练、评估、检查、联调启动）。
- `docker/`：容器化相关文件。
- `docs/`：项目文档。

## 3. 关键路径规范
- 训练输出默认目录：`artifacts/models/`
- 评估指标默认目录：`artifacts/metrics/`
- 在线推理模型路径：由 `configs/model.yaml` 的 `model_path` 控制。
- Go 工作区：`go.work`（根目录，指向 `./gateway`，使 `go run/test/build ./gateway/...` 可在根目录执行）。
- 网关配置路径：`configs/gateway.yaml`（模板：`configs/gateway.yaml.example`）。
- 对外接口契约路径：`shared/schemas/openapi.yaml`（网关、前端、推理服务字段以此对齐）。

## 4. 常用命令
- 安装依赖：`uv sync`
- 安装前端依赖：`bun --cwd frontend install`
- 启动服务：`uv run uvicorn app.main:app --reload`
- 启动 Go 网关：`go run ./gateway/cmd/gateway`
- 启动前端（Web）：`bun --cwd frontend run dev`
- 启动前端（Desktop）：`bun --cwd frontend run tauri:dev`
- 运行测试：`uv run pytest -q`
- 运行 Go 测试：`go test ./gateway/...`
- 前端类型检查：`bun --cwd frontend run typecheck`
- 前端测试：`bun --cwd frontend run test`
- 前端构建（CSR/SSG）：`bun --cwd frontend run generate`
- 训练模型：`uv run python training/train.py --data path/to/data.yaml --model yolo26n.pt`
- 评估模型：`uv run python training/eval.py --model artifacts/models/<exp>/weights/best.pt --data path/to/data.yaml`

## 5. 脚本入口（优先使用）
- `sh scripts/app.sh --host 127.0.0.1 --port 8000`
- `sh scripts/gateway.sh --config configs/gateway.yaml`
- `sh scripts/stack.sh --app-host 127.0.0.1 --app-port 8000 --gateway-config configs/gateway.yaml`
- `sh scripts/frontend.sh --host 127.0.0.1 --port 3000`
- `sh scripts/desktop.sh`
- `sh scripts/train.sh --data data/lichi/data.yaml --name lychee_v1`
- `sh scripts/eval.sh --data data/lichi/data.yaml --exp lychee_v1`
- `sh scripts/verify.sh`
- `powershell -ExecutionPolicy Bypass -File scripts/app.ps1 -Host 127.0.0.1 -Port 8000`
- `powershell -ExecutionPolicy Bypass -File scripts/gateway.ps1 -Config configs/gateway.yaml`
- `powershell -ExecutionPolicy Bypass -File scripts/stack.ps1 -AppHost 127.0.0.1 -AppPort 8000 -GatewayConfig configs/gateway.yaml`
- `powershell -ExecutionPolicy Bypass -File scripts/frontend.ps1 -Host 127.0.0.1 -Port 3000`
- `powershell -ExecutionPolicy Bypass -File scripts/desktop.ps1`
- `powershell -ExecutionPolicy Bypass -File scripts/train.ps1 -Data data/lichi/data.yaml -Name lychee_v1`
- `powershell -ExecutionPolicy Bypass -File scripts/eval.ps1 -Data data/lichi/data.yaml -Exp lychee_v1`
- `powershell -ExecutionPolicy Bypass -File scripts/verify.ps1`

## 6. 开发与改动规则
- 优先最小改动：只改与任务直接相关的文件。
- 不要擅自重命名顶层目录（如 `app/`、`gateway/`、`training/`、`tests/`）。
- 修改配置或路径时，同步检查：
  - `README.md` 示例命令
  - `configs/*.yaml.example`
  - 相关测试
- 新增共享字段时，优先更新 `shared/` 下定义并保持前后端一致。
- 前端默认只调用 `gateway/`，不直连 `app/`。
- 任何会影响行为的改动，至少运行一次相关测试。

## 7. 数据与版本信息
- 数据集引用：
  - Zhiqing, Zhao (2025), "lichi-maturity", Mendeley Data, V1, doi: `10.17632/c3rk9gv4w9.1`
- 本仓库不提交原始数据与大模型权重（遵循 `.gitignore`）。

## 8. 提交前检查清单
- 命令和路径示例是否可运行。
- 是否引入了新的硬编码绝对路径。
- 是否破坏 `configs/*.yaml.example` 的可用性。
- 是否保留类别映射一致性（green/half/red/young）。
- `shared/schemas/openapi.yaml` 与 `frontend/`、`gateway/`、`app/` 的字段是否一致。
- `frontend` 的类型检查/测试/构建是否通过（`typecheck`、`test`、`generate`）。
- 前端调用路径是否仍为 `frontend -> gateway -> app`。
- 前端颜色与标签映射是否与 `shared/constants/ripeness.json` 一致。
- 测试是否通过，或已明确说明未执行原因。

## 9. 不确定时的默认策略
- 选择保守方案：优先保证现有 API 与训练流程兼容。
- 若需要结构性重构，先在 PR/变更说明里给出迁移步骤，再实施。
