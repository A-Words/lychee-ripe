# AGENTS.md

## 1. 文档定位

- 本文档面向自动化 Agent，提供当前 Monorepo 的事实约束、默认工作流和提交前检查。
- 先对齐目录职责，再改代码，再做验证。
- 权威定义：
  - 接口与字段以 `shared/contracts/schemas/openapi.yaml` 为准。
  - 成熟度映射以 `shared/contracts/constants/ripeness.json` 为准。
  - 产品需求与范围以 `docs/prd.md` 为准。

## 2. 项目快照

- 项目目标：荔枝目标检测与成熟度识别，提供后端推理、网关治理与前端可视化。
- 调用链路：`clients/orchard-console -> services/gateway -> services/inference-api`。
- 网关实现：Go HTTP 反向代理 + WebSocket 透传，当前不是 gRPC。
- 成熟度类别映射：`0=green`，`1=half`，`2=red`，`3=young`。

当前顶层职责分组：

- `clients/orchard-console/`：Nuxt Web + Tauri Desktop
- `services/inference-api/`：FastAPI 推理服务、Python 测试、Python 锁文件
- `services/gateway/`：Go 网关（鉴权、限流、日志、代理）
- `shared/contracts/`：共享常量与 OpenAPI 契约
- `shared/python/`：训练与推理共用 Python helper
- `mlops/training/`：训练与评估脚本
- `mlops/data/`：数据集与标注
- `mlops/artifacts/`：模型、指标、日志与本地 sqlite 数据
- `mlops/pretrained/`：预训练权重
- `tooling/configs/`：配置模板与本地配置
- `tooling/scripts/`：脚本入口
- `tooling/docker/`：容器构建文件
- `tests/stack/`：跨服务 smoke 测试

关键路径：

- 训练输出：`mlops/artifacts/models/`
- 评估输出：`mlops/artifacts/metrics/`
- 网关本地数据库：`mlops/artifacts/data/gateway.db`
- 预训练权重：`mlops/pretrained/yolo26n.pt`
- 推理模型配置：`tooling/configs/model.yaml`
- 网关配置：`tooling/configs/gateway.yaml`
- 对外契约：`shared/contracts/schemas/openapi.yaml`

## 3. 执行工作流

### 3.1 依赖与配置准备

- 安装 JS 依赖：`bun install`
- 安装 Python 依赖：`cd services/inference-api && uv sync`
- 准备本地配置：
  - `tooling/configs/model.yaml.example -> tooling/configs/model.yaml`
  - `tooling/configs/service.yaml.example -> tooling/configs/service.yaml`
  - `tooling/configs/gateway.yaml.example -> tooling/configs/gateway.yaml`
- 锁切换：
  - `sh tooling/scripts/switch-lock.sh --target cpu|cu128|auto`
  - `powershell -ExecutionPolicy Bypass -File tooling/scripts/switch-lock.ps1 -Target cpu|cu128|auto`

### 3.2 服务启动

- 根入口：
  - `bun run dev`
  - `bun run dev:inference-api`
  - `bun run dev:gateway`
  - `bun run dev:orchard-console`
- 分服务直启：
  - Inference API：`cd services/inference-api && uv run python -m uvicorn app.main:app --reload --host 127.0.0.1 --port 8000`
  - Gateway：`go run ./services/gateway/cmd/gateway --config tooling/configs/gateway.yaml`
  - Orchard Console Web：`cd clients/orchard-console && bun run dev -- --host 127.0.0.1 --port 3000`
  - Orchard Console Desktop：`cd clients/orchard-console && bun run tauri:dev`
- 脚本入口：
  - `tooling/scripts/inference-api.*`
  - `tooling/scripts/gateway.*`
  - `tooling/scripts/orchard-console.*`
  - `tooling/scripts/desktop.*`
  - `tooling/scripts/stack.*`

### 3.3 训练与评估

- Workspace 便捷入口：
  - `bun run --filter @lychee-ripe/training train`
  - `bun run --filter @lychee-ripe/training eval`
  - 以上默认使用 `mlops/data/lichi/data.yaml` 与 `lychee_v1` 相关产物；需要覆盖时，用 `--` 继续追加参数
  - workspace 默认路径按 repo-root 相对解释；fresh clone 下即使 `mlops/artifacts/` 尚不存在，输出也必须落在仓库内
- 训练：
  - `uv run --project services/inference-api python mlops/training/train.py --data mlops/data/lichi/data.yaml --model mlops/pretrained/yolo26n.pt --project mlops/artifacts/models --name lychee_v1`
- 评估：
  - `uv run --project services/inference-api python mlops/training/eval.py --model mlops/artifacts/models/lychee_v1/weights/best.pt --data mlops/data/lichi/data.yaml --output mlops/artifacts/metrics/lychee_v1-eval_metrics.json`
- 脚本入口：
  - `tooling/scripts/train.*`
  - `tooling/scripts/eval.*`

### 3.4 校验与测试

- Python：`cd services/inference-api && uv run python -m pytest -q`
- Go：`go test ./services/gateway/...`
- Frontend：
  - `bun run --filter @lychee-ripe/orchard-console typecheck`
  - `bun run --filter @lychee-ripe/orchard-console test`
  - `bun run --filter @lychee-ripe/orchard-console generate`
- 根入口：
  - `bun run verify`
  - `bun run test:stack`

### 3.5 容器

- Compose 入口：`docker-compose.yml`
- Inference API 构建文件：`tooling/docker/Dockerfile.inference-api`
- Gateway 构建文件：`tooling/docker/Dockerfile.gateway`

## 4. 改动规则

- 优先最小改动，只改与当前任务直接相关的文件。
- 保持职责边界清晰：
  - 客户端代码放在 `clients/`
  - 服务代码放在 `services/`
  - 契约与共享常量放在 `shared/contracts/`
  - 训练生命周期资产放在 `mlops/`
  - 工程脚本、配置、Docker 放在 `tooling/`
  - 跨服务测试放在 `tests/stack/`
- 不重新引入旧的 `apps/`、`packages/`、根平铺 `configs/`、`scripts/`、`docker/`、`data/`、`artifacts/` 布局，除非用户明确要求。
- 改配置或路径时，必须同步检查：
  - `README.md`
  - `AGENTS.md`
  - `tooling/configs/*.yaml.example`
  - `tooling/scripts/*`
  - `docker-compose.yml` 与 `tooling/docker/*`
- 改 Python 依赖时：
  - 同步更新 `services/inference-api/uv.lock.cpu` 与 `services/inference-api/uv.lock.cu128`
  - 提交前恢复 `services/inference-api/uv.lock` 为 CPU 基线
- 新增共享字段时，优先在 `shared/contracts/` 定义，再同步 `services/inference-api`、`services/gateway`、`clients/orchard-console`
- 前端禁止直连 `services/inference-api`，默认必须走 `services/gateway`
- 任何行为改动，至少运行一次相关测试；若未执行，必须说明原因和风险
- `tooling/configs/gateway.yaml.example` 应保持本地直启可用，默认 `upstream.base_url` 指向 `http://127.0.0.1:8000`
- Docker Compose 应使用独立的 `tooling/configs/gateway.compose.yaml`，其中 `upstream.base_url` 指向容器服务名 `http://inference-api:8000`

## 5. 提交前检查

- 命令与路径示例可运行，且与脚本实现一致
- 未引入硬编码绝对路径
- 未破坏 `tooling/configs/*.yaml.example` 可用性
- 目录职责未被破坏，没有把工程资产重新平铺回根目录
- 类别映射保持一致：`green/half/red/young`
- 接口契约一致性：
  - `shared/contracts/schemas/openapi.yaml` 与 `services/inference-api`、`services/gateway`、`clients/orchard-console` 字段一致
- 前端质量门禁通过：`typecheck`、`test`、`generate`
- 调用链未偏离：`clients/orchard-console -> services/gateway -> services/inference-api`
- 前端颜色与标签映射与 `shared/contracts/constants/ripeness.json` 一致
- 测试执行情况已记录

## 6. 默认策略

- 不确定时优先兼容现有 API、训练流程、脚本入口和根命令名
- 涉及结构性改动时，先明确影响范围，再实施变更
- 若发现旧路径残留，优先修正为当前职责分组结构，而不是补兼容层
