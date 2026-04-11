# Lychee Ripe

荔枝目标检测与成熟度识别 Monorepo。目录按职责分组为 `clients/`、`services/`、`shared/`、`mlops/`、`tooling/`、`tests/`，固定调用链路为 `clients/orchard-console -> services/gateway -> services/inference-api`。

## Overview

- `clients/orchard-console`：Nuxt Web + Tauri Desktop
- `services/gateway`：Go 网关，负责鉴权、授权、限流、日志、代理与 WebSocket 透传
- `services/inference-api`：FastAPI 推理服务
- `shared/contracts`：OpenAPI 契约与成熟度常量
- `shared/python`：训练与推理共用 Python helper
- `mlops/training`：训练与评估脚本
- `mlops/data`：数据集与标注
- `mlops/artifacts`：模型、指标、日志与本地 sqlite 数据
- `tooling/configs`：配置模板与本地配置
- `tests/stack`：跨服务 smoke 测试

权威文件：

- [shared/contracts/schemas/openapi.yaml](shared/contracts/schemas/openapi.yaml)
- [shared/contracts/constants/ripeness.json](shared/contracts/constants/ripeness.json)
- [docs/prd.md](docs/prd.md)

成熟度映射固定为 `0=green`、`1=half`、`2=red`、`3=young`。

## Prerequisites

- Python `>= 3.11`
- `uv`
- Bun `>= 1.2`
- Go，版本以 [services/gateway/go.mod](services/gateway/go.mod) 为准

## Quick Start

安装依赖：

```sh
bun install
uv sync --project services/inference-api --extra cpu
```

准备本地配置：

```sh
cp tooling/configs/model.yaml.example tooling/configs/model.yaml
cp tooling/configs/service.yaml.example tooling/configs/service.yaml
cp tooling/configs/gateway.yaml.example tooling/configs/gateway.yaml
```

Windows PowerShell：

```powershell
Copy-Item tooling/configs/model.yaml.example tooling/configs/model.yaml
Copy-Item tooling/configs/service.yaml.example tooling/configs/service.yaml
Copy-Item tooling/configs/gateway.yaml.example tooling/configs/gateway.yaml
```

启动联调栈：

```sh
bun run dev
```

Python 相关根入口默认使用 `cpu`。如需切到 CUDA 12.8，统一追加 `-- --target cu128`，例如：

```sh
bun run dev:inference-api -- --target cu128
bun run test -- --target cu128
```

## Common Commands

根入口：

```sh
bun run dev
bun run dev:inference-api
bun run dev:gateway
bun run dev:orchard-console
bun run build
bun run generate
bun run typecheck
bun run test
bun run verify
bun run test:stack
```

分服务直启：

```sh
uv run --project services/inference-api --extra cpu python -m uvicorn app.main:app --reload --host 127.0.0.1 --port 8000
go run ./services/gateway/cmd/gateway --config tooling/configs/gateway.yaml
bun run --cwd clients/orchard-console dev -- --host 127.0.0.1 --port 3000
bun run --cwd clients/orchard-console tauri:dev
```

Windows 上如果希望避免 `go run` 每次生成临时二进制触发防火墙重复提示，使用：

```sh
bun run dev:gateway
```

训练与评估：

```sh
bun run --filter @lychee-ripe/training train
bun run --filter @lychee-ripe/training eval
uv run --project services/inference-api --extra cpu python mlops/training/train.py --data mlops/data/lichi/data.yaml --model mlops/pretrained/yolo26n.pt --project mlops/artifacts/models --name lychee_v1
uv run --project services/inference-api --extra cpu python mlops/training/eval.py --model mlops/artifacts/models/lychee_v1/weights/best.pt --data mlops/data/lichi/data.yaml --output mlops/artifacts/metrics/lychee_v1-eval_metrics.json
```

默认产物位置：

- 模型：`mlops/artifacts/models/`
- 指标：`mlops/artifacts/metrics/`
- 数据库：`mlops/artifacts/data/gateway.db`
- 预训练权重：`mlops/pretrained/`

## Verification

统一入口：

```sh
bun run test
bun run verify
bun run test:stack
```

单项执行：

```sh
uv run --project services/inference-api --extra cpu python -m pytest -q
go test ./services/gateway/...
bun run --filter @lychee-ripe/orchard-console typecheck
bun run --filter @lychee-ripe/orchard-console test
bun run --filter @lychee-ripe/orchard-console generate
```

`bun run verify` 会额外执行：

- `@lychee-ripe/contracts#verify`
- `@lychee-ripe/python-shared#verify`

`bun run test:stack` 会自动拉起 `bun run dev`，等待 `frontend -> gateway -> api` 就绪后执行 smoke 测试，结束后自动清理进程。

## Turbo Conventions

- `build`、`test`、`typecheck`、`generate`、`verify` 默认可缓存
- `dev`、`train`、`eval` 默认不缓存
- `verify` 是 Turbo 聚合任务，不再依赖包内 shell `&&` 串联
- `clients/orchard-console` 的相关任务会将 `NUXT_PUBLIC_*` 纳入 hash
- Python 相关任务会将 `LYCHEE_PY_TARGET` 纳入 hash，`cpu` 与 `cu128` 缓存隔离
- `shared/contracts` 与 `shared/python` 的变化会触发依赖任务重算
- 工程工具缓存统一收口到根目录 `.cache/`，当前包括 `go-build`、`uv`、`xdg`、`torchinductor`

远程缓存按环境变量启用，仓库内不写死平台信息：

```sh
TURBO_TEAM=your-team
TURBO_TOKEN=your-token
bun run verify
```

自建 Remote Cache 再额外提供：

```sh
TURBO_API=https://your-cache.example.com
```

## Config Notes

核心配置文件：

- `tooling/configs/model.yaml`
- `tooling/configs/service.yaml`
- `tooling/configs/gateway.yaml`
- `tooling/configs/gateway.compose.yaml`

Gateway 约定：

- `trace.mode=database|blockchain`，默认 `database`
- `auth.mode=disabled|oidc`，默认 `disabled`
- `disabled` 模式会为受保护接口注入模拟 `admin`
- `oidc` 模式下，Web 端走 Gateway 托管的 OIDC 登录与 HttpOnly Cookie，Tauri 继续使用 Bearer Token
- 首次在空库上启用 OIDC 时，必须提供 `auth.bootstrap_admin_email` 或 `LYCHEE_AUTH_BOOTSTRAP_ADMIN_EMAIL`
- Web Cookie 默认 `SameSite=Lax`，跨站部署必须改成 `cookie_same_site=none` 且同时开启 `cookie_secure=true`
- 首次绑定预创建用户时，网关要求 `access_token` 自带 `email` claim，不额外调用 `userinfo`

本地开发常用环境变量：

```sh
LYCHEE_AUTH_MODE=disabled
NUXT_PUBLIC_AUTH_MODE=disabled
LYCHEE_SEED_DEFAULT_RESOURCES_ENABLED=true
```

## Docker

```sh
docker compose up --build
```

相关文件：

- `docker-compose.yml`
- `tooling/docker/Dockerfile.inference-api`
- `tooling/docker/Dockerfile.gateway`

Compose 默认让 Gateway 读取 `tooling/configs/gateway.compose.yaml`，其中上游地址指向 `http://inference-api:8000`。

## Notes

- 前端默认必须经由 Gateway 访问 Inference API，不应直连 `services/inference-api`
- 新增共享字段时，优先更新 `shared/contracts`，再同步到各服务
- `shared/python/lychee_common` 仅承载跨训练与推理共用的 Python 工具
- 自动化 Agent 约束见 [AGENTS.md](AGENTS.md)
