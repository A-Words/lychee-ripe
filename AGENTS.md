# AGENTS.md

## 1. 文档定位

- 本文档面向自动化 Agent，定义当前 Monorepo 的事实约束、默认工作流与提交前检查。
- 先对齐目录职责，再改代码，再做验证。
- 权威定义：
  - 接口与字段以 `shared/contracts/schemas/openapi.yaml` 为准
  - 成熟度映射以 `shared/contracts/constants/ripeness.json` 为准
  - 产品需求与范围以 `docs/prd.md` 为准

## 2. 项目事实

- 项目目标：荔枝目标检测与成熟度识别，提供推理服务、网关治理与前端可视化
- 固定调用链路：`clients/orchard-console -> services/gateway -> services/inference-api`
- 网关实现：Go HTTP 反向代理 + WebSocket 透传，不是 gRPC
- 成熟度映射：`0=green`、`1=half`、`2=red`、`3=young`

顶层职责：

- `clients/orchard-console/`：Nuxt Web + Tauri Desktop
- `services/inference-api/`：FastAPI 推理服务、Python 测试、Python 锁文件
- `services/gateway/`：Go 网关
- `shared/contracts/`：共享常量与 OpenAPI 契约
- `shared/python/`：训练与推理共用 Python helper
- `mlops/training/`：训练与评估脚本
- `mlops/data/`：数据集与标注
- `mlops/artifacts/`：模型、指标、日志与本地 sqlite 数据
- `mlops/pretrained/`：预训练权重
- `tooling/configs/`：配置模板与本地配置
- `tooling/scripts/`：Turbo、工作区 verify、训练、评估与缓存环境脚本
- `tooling/docker/`：容器构建文件
- `tests/stack/`：跨服务 smoke 测试

关键路径：

- 训练输出：`mlops/artifacts/models/`
- 评估输出：`mlops/artifacts/metrics/`
- 网关数据库：`mlops/artifacts/data/gateway.db`
- 预训练权重：`mlops/pretrained/yolo26n.pt`
- 模型配置：`tooling/configs/model.yaml`
- 网关配置：`tooling/configs/gateway.yaml`
- Compose 网关配置：`tooling/configs/gateway.compose.yaml`

Gateway 运行语义：

- `trace.mode=database|blockchain`，默认 `database`
- `database` 模式下主状态为 `stored`，不初始化链适配器
- `blockchain` 模式下保留 `pending_anchor / anchored / anchor_failed` 与补链能力
- `auth.mode=disabled|oidc`，默认 `disabled`
- `disabled` 模式会为受保护接口注入模拟 `admin`
- `oidc` 模式下，Web 端走 Gateway 托管的 OIDC 授权码交换与 HttpOnly Session Cookie，Tauri 继续使用 `JWT + JWKS` Bearer Token
- 首次在空库启用 OIDC 时，必须提供 `auth.bootstrap_admin_email` 或 `LYCHEE_AUTH_BOOTSTRAP_ADMIN_EMAIL`
- 首次绑定预创建用户时，网关要求 `access_token` 自带 `email` claim，不额外调用 `userinfo`
- Web Cookie 默认 `SameSite=Lax`，跨站部署必须改成 `auth.web.cookie_same_site=none` 且同时开启 `auth.web.cookie_secure=true`

## 3. 标准工作流

### 3.1 依赖与配置

- JS 依赖：`bun install`
- Python 依赖：`uv sync --project services/inference-api --extra cpu`
- CUDA 12.8：`uv sync --project services/inference-api --extra cu128`
- 本地配置：
  - `tooling/configs/model.yaml.example -> tooling/configs/model.yaml`
  - `tooling/configs/service.yaml.example -> tooling/configs/service.yaml`
  - `tooling/configs/gateway.yaml.example -> tooling/configs/gateway.yaml`

### 3.2 启动

- 根入口：
  - `bun run dev`
  - `bun run dev:inference-api`
  - `bun run dev:gateway`
  - `bun run dev:orchard-console`
- Python 相关根入口默认显式使用 `cpu`；切到 CUDA 12.8 时统一追加 `-- --target cu128`
- `bun run dev` 以 `@lychee-ripe/orchard-console` 为入口，再由 Turbo 任务图联动带起 `gateway` 与 `inference-api`
- 分服务直启：
  - `uv run --project services/inference-api --extra cpu python -m uvicorn app.main:app --reload --host 127.0.0.1 --port 8000`
  - `go run ./services/gateway/cmd/gateway --config tooling/configs/gateway.yaml`
  - `bun run --cwd clients/orchard-console dev -- --host 127.0.0.1 --port 3000`
  - `bun run --cwd clients/orchard-console tauri:dev`
- Windows 上如果要避免 `go run` 触发重复防火墙提示，使用 `bun run dev:gateway`

### 3.3 训练与评估

- Workspace 入口：
  - `bun run --filter @lychee-ripe/training train`
  - `bun run --filter @lychee-ripe/training eval`
- 默认使用 `mlops/data/lichi/data.yaml` 与 `lychee_v1` 产物；覆盖参数时通过 `--` 继续传递
- 切换 Python 后端时使用 `-- --target cpu|cu128`
- 直接运行：
  - `uv run --project services/inference-api --extra cpu python mlops/training/train.py --data mlops/data/lichi/data.yaml --model mlops/pretrained/yolo26n.pt --project mlops/artifacts/models --name lychee_v1`
  - `uv run --project services/inference-api --extra cpu python mlops/training/eval.py --model mlops/artifacts/models/lychee_v1/weights/best.pt --data mlops/data/lichi/data.yaml --output mlops/artifacts/metrics/lychee_v1-eval_metrics.json`
- 脚本入口只保留：
  - `tooling/scripts/train.*`
  - `tooling/scripts/eval.*`

### 3.4 校验与测试

- 根入口：
  - `bun run test`
  - `bun run verify`
  - `bun run test:stack`
- 单项执行：
  - `uv run --project services/inference-api --extra cpu python -m pytest -q`
  - `go test ./services/gateway/...`
  - `bun run --filter @lychee-ripe/orchard-console typecheck`
  - `bun run --filter @lychee-ripe/orchard-console test`
  - `bun run --filter @lychee-ripe/orchard-console generate`
- `bun run verify` 会额外执行 `@lychee-ripe/contracts#verify` 与 `@lychee-ripe/python-shared#verify`
- `bun run verify` 还必须校验 `tooling/configs/{model,service,gateway}.yaml.example` 存在
- 直接执行 `bun run --filter <workspace> verify` 时，各 workspace 通过 `tooling/scripts/workspace-verify.mjs` 运行本包真实校验；根 `bun run verify` 仍由 Turbo 聚合
- `bun run test:stack` 会自动拉起联调链路并在结束后清理进程

### 3.5 Turbo 约定

- `build`、`test`、`typecheck`、`generate`、`verify` 默认可缓存
- `dev`、`train`、`eval` 默认不缓存
- package 级 `turbo.json` 优先使用 `$TURBO_DEFAULT$`，再补充共享输入
- 前端任务显式声明 `NUXT_PUBLIC_*`
- Python 任务显式声明 `LYCHEE_PY_TARGET`
- `shared/contracts` 与 `shared/python/lychee_common` 的变化必须触发依赖任务重算
- `verify` 是 Turbo 聚合任务，不再依赖包内 shell 串联
- `bun run dev` 虽然只过滤 `@lychee-ripe/orchard-console`，但会通过 Turbo 的 `with` 关系联动带起 `gateway` 与 `inference-api`
- 远程缓存保持平台中立，通过 `TURBO_TOKEN`、`TURBO_TEAM`、可选 `TURBO_API` 在环境中开启
- 工程工具缓存统一收口到根 `.cache/`，当前包括 `go-build`、`uv`、`xdg`、`torchinductor`

### 3.6 容器

- Compose 入口：`docker-compose.yml`
- Dockerfile：
  - `tooling/docker/Dockerfile.inference-api`
  - `tooling/docker/Dockerfile.gateway`
- Compose 中 Gateway 应使用 `tooling/configs/gateway.compose.yaml`，其 `upstream.base_url` 指向 `http://inference-api:8000`

## 4. 改动规则

- 优先最小改动，只改与当前任务直接相关的文件
- 保持职责边界清晰，不重新引入旧的根平铺目录结构
- 改配置或路径时，必须同步检查：
  - `README.md`
  - `AGENTS.md`
  - `tooling/configs/*.yaml.example`
  - `tooling/scripts/*`
  - `docker-compose.yml`
  - `tooling/docker/*`
- 改 Python 依赖时，仅维护 `services/inference-api/uv.lock`
- `torch` / `torchvision` 的 CPU/CUDA 选择通过 `project.optional-dependencies` 与 `tool.uv.sources` 管理
- Python 依赖变更后至少验证 `uv sync --extra cpu --frozen`；涉及 CUDA 时再补 `uv sync --extra cu128 --frozen`
- 新增共享字段时，优先在 `shared/contracts/` 定义，再同步到各服务
- 前端禁止直连 `services/inference-api`，默认必须走 `services/gateway`
- 任何行为改动至少运行一次相关测试；未执行时必须说明原因和风险
- `tooling/configs/gateway.yaml.example` 应保持本地直启可用，默认 `upstream.base_url` 指向 `http://127.0.0.1:8000`
- 网关代码默认不应向空库自动写入演示果园或地块；如需本地演示数据，应通过 `seed.default_resources_enabled` 显式开启
- `tooling/configs/gateway.yaml.example` 默认 `auth.mode=disabled`

## 5. 提交前检查

- 命令与路径示例可运行，且与脚本实现一致
- 未引入硬编码绝对路径
- 未破坏 `tooling/configs/*.yaml.example` 可用性
- 目录职责未被破坏
- 类别映射保持一致：`green/half/red/young`
- 接口契约一致性保持成立
- 前端质量门禁通过：`typecheck`、`test`、`generate`
- 调用链未偏离：`clients/orchard-console -> services/gateway -> services/inference-api`
- 测试执行情况已记录

## 6. 默认策略

- 不确定时优先兼容现有 API、训练流程、脚本入口与根命令名
- 涉及结构性改动时，先明确影响范围，再实施变更
- 发现旧路径残留时，优先修正为当前职责分组结构，而不是补兼容层
