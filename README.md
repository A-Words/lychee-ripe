# Lychee Ripe

荔枝目标检测与成熟度识别 Monorepo，按职责分组为 `clients/`、`services/`、`shared/`、`mlops/`、`tooling/`、`tests/`。

## Overview

- `clients/orchard-console`：Nuxt Web + Tauri Desktop，果园业务操作前端
- `services/inference-api`：FastAPI 推理服务
- `services/gateway`：Go 网关，负责 OIDC 鉴权、本地授权、限流、日志、代理与 WebSocket 透传，并提供 `database / blockchain` 双溯源模式
- `shared/contracts`：共享常量与 OpenAPI 契约
- `shared/python`：训练与推理共用的 Python helper
- `mlops/training`：训练与评估脚本
- `mlops/data`：数据集与标注
- `mlops/artifacts`：模型、指标、日志与网关本地数据库
- `tooling/configs`：配置模板与本地配置
- `tooling/scripts`：训练、评估、缓存环境与跨服务联调脚本
- `tooling/docker`：镜像构建文件
- `tests/stack`：跨服务 smoke 测试

调用链路固定为 `frontend -> gateway -> api`。

Gateway 溯源模式：

- `trace.mode=database`：默认模式，批次以数据库存证为主，不初始化链适配器、不执行补链或链上校验
- `trace.mode=blockchain`：启用链上锚定、补链与公开验真能力

Gateway 认证模式：

- `auth.mode=disabled`：开发旁路，所有受保护接口默认以模拟 `admin` 身份放行
- `auth.mode=oidc`：Web 端通过 Gateway 托管的 OIDC 授权码交换与 HttpOnly Session Cookie 登录，Tauri/原生端继续使用 Bearer Access Token；角色与启停状态由本地数据库维护

成熟度映射固定为：

- `0 = green`
- `1 = half`
- `2 = red`
- `3 = young`

权威文件：

- 常量映射：[shared/contracts/constants/ripeness.json](shared/contracts/constants/ripeness.json)
- 接口契约：[shared/contracts/schemas/openapi.yaml](shared/contracts/schemas/openapi.yaml)
- 产品需求：[docs/prd.md](docs/prd.md)

## Prerequisites

- Python `>= 3.11`
- `uv`
- Bun `>= 1.2`
- Go（见 [services/gateway/go.mod](services/gateway/go.mod)）

## Setup

安装依赖：

```sh
bun install
cd services/inference-api && uv sync --extra cpu
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

Python 加速后端通过 `uv` extra 选择：

```sh
cd services/inference-api && uv sync --extra cpu
```

```powershell
cd services/inference-api; uv sync --extra cu128
```

## Run

根入口：

```sh
bun run dev
bun run dev:inference-api
bun run dev:gateway
bun run dev:orchard-console
```

这些根入口默认显式使用 `cpu`；如需切到 CUDA 12.8，可追加 `-- --target cu128`，例如：

```sh
bun run dev:inference-api -- --target cu128
bun run test -- --target cu128
bun run verify -- --target cpu
```

`bun run dev` 现在以 `@lychee-ripe/orchard-console` 为入口，再由 Turbo 任务图联动带起 `gateway` 与 `inference-api` 的 `dev` 任务；不再在根脚本里手工拼接并行过滤参数。

`LYCHEE_PY_TARGET` 只会进入 Python-backed Turbo task 的缓存键，CPU 和 CUDA 的 `test` / `verify` 结果不会混用；非 Python workspace 继续复用跨 target 缓存。

分服务直接启动：

```sh
cd services/inference-api && uv run --extra cpu python -m uvicorn app.main:app --reload --host 127.0.0.1 --port 8000
go run ./services/gateway/cmd/gateway --config tooling/configs/gateway.yaml
cd clients/orchard-console && bun run dev -- --host 127.0.0.1 --port 3000
cd clients/orchard-console && bun run tauri:dev
```

跨服务脚本入口：

```sh
sh tooling/scripts/stack.sh --target cpu --app-host 127.0.0.1 --app-port 8000 --gateway-config tooling/configs/gateway.yaml --frontend-host 127.0.0.1 --frontend-port 3000
```

## Training And Eval

workspace 便捷入口：

```sh
bun run --filter @lychee-ripe/training train
bun run --filter @lychee-ripe/training eval
```

这两个命令默认使用 `mlops/data/lichi/data.yaml` 和 `lychee_v1` 产物；如需覆盖，继续通过 `--` 追加参数，例如：

```sh
bun run --filter @lychee-ripe/training train -- --target cpu --data mlops/data/lichi/data.yaml --name custom_run
bun run --filter @lychee-ripe/training eval -- --target cu128 --model mlops/artifacts/models/custom_run/weights/best.pt --data mlops/data/lichi/data.yaml --output mlops/artifacts/metrics/custom_run.json
```

training workspace 入口默认显式使用 `cpu`；如需切到 CUDA 12.8，统一通过 `-- --target cu128` 覆写。

workspace 默认参数和输出目录都按 repo-root 相对路径解释；即使 fresh clone 时 `mlops/artifacts/` 还不存在，产物也会创建在仓库内。

直接运行：

```sh
uv run --project services/inference-api --extra cpu python mlops/training/train.py --data mlops/data/lichi/data.yaml --model mlops/pretrained/yolo26n.pt --project mlops/artifacts/models --name lychee_v1
uv run --project services/inference-api --extra cpu python mlops/training/eval.py --model mlops/artifacts/models/lychee_v1/weights/best.pt --data mlops/data/lichi/data.yaml --output mlops/artifacts/metrics/lychee_v1-eval_metrics.json
```

脚本入口：

```sh
sh tooling/scripts/train.sh --target cpu --data mlops/data/lichi/data.yaml --name lychee_v1
sh tooling/scripts/eval.sh --target cpu --data mlops/data/lichi/data.yaml --exp lychee_v1
```

默认产物位置：

- 模型：`mlops/artifacts/models/`
- 指标：`mlops/artifacts/metrics/`
- 预训练权重：`mlops/pretrained/`

## Verification

统一入口：

```sh
bun run test
bun run verify
```

`bun run verify` 现在除了业务 workspace，还会执行：

- `@lychee-ripe/contracts#verify`：校验 `openapi.yaml` 可解析且成熟度映射结构合法
- `@lychee-ripe/python-shared#verify`：校验 `shared/python` 元数据与 `lychee_common` 可导入

Turbo 任务语义约定：

- `dev`、`train`、`eval` 不缓存
- `build`、`test`、`typecheck`、`generate`、`verify` 走 Turbo 缓存
- `clients/orchard-console` 的 `build/generate/typecheck/test/dev` 会把 `NUXT_PUBLIC_*` 变量纳入 hash
- `services/inference-api` 与 `mlops/training` 的 Python 任务会把 `LYCHEE_PY_TARGET` 纳入 hash
- `shared/contracts` 与 `shared/python` 的源码变化会触发依赖它们的前端、推理和训练任务重新计算

单独执行：

```sh
cd services/inference-api && uv run --extra cpu python -m pytest -q
go test ./services/gateway/...
bun run --filter @lychee-ripe/orchard-console typecheck
bun run --filter @lychee-ripe/orchard-console test
bun run --filter @lychee-ripe/orchard-console generate
```

跨服务 smoke：

```sh
bun run test:stack
```

`test:stack` 需要先把 API、Gateway、Frontend 启起来，再验证 `3000 -> 9000 -> api` 的基础联通。

不再维护单独的 `tooling/scripts/verify.*`；统一校验入口以 `bun run verify` 为准。

## Turbo And Remote Cache

仓库采用 Turborepo 统一编排 `build/test/typecheck/generate/verify/dev/train/eval`，并显式声明跨语言输入：

- 前端任务额外跟踪 `shared/contracts/{constants,schemas}`，确保契约或成熟度映射变化会触发重算
- 推理服务与训练任务额外跟踪 `shared/python/lychee_common`、共享契约，以及相关 `tooling/configs/*.yaml`
- `verify` 是 Turbo 聚合任务；包内不再通过 shell `&&` 手工串联子任务

平台中立远程缓存接入方式：

```sh
TURBO_TEAM=your-team
TURBO_TOKEN=your-token
bun run verify
```

如果使用自建 Remote Cache，再额外提供：

```sh
TURBO_API=https://your-cache.example.com
```

仓库内不会写死 team、token 或平台专属配置；是否启用远程缓存由运行环境中的 `TURBO_TOKEN` / `TURBO_TEAM` / `TURBO_API` 决定。

仓库自带脚本会把工程工具缓存统一收口到根目录 `.cache/`，当前包括 `go-build`、`uv`、`xdg`、`torchinductor`；`mlops/artifacts/` 只保留模型、指标、日志与数据库等业务产物。

## Config

- 推理模型配置：`tooling/configs/model.yaml`
- 服务配置：`tooling/configs/service.yaml`
- 网关配置：`tooling/configs/gateway.yaml`

默认模型路径、网关默认配置与本地 sqlite 路径都已改成 repo-root 相对解析，直接从仓库根或各服务目录运行都可以。
`tooling/configs/gateway.yaml.example` 默认面向本地直启，`upstream.base_url` 使用 `http://127.0.0.1:8000`。
网关代码默认不会向空库自动注入演示果园/地块；只有当 `seed.default_resources_enabled=true` 或 `LYCHEE_SEED_DEFAULT_RESOURCES_ENABLED=true` 时才会写入示例资源。仓库提供的 `gateway.yaml.example` 为了本地演示默认开启这一选项，而 `gateway.compose.yaml` 默认关闭，避免部署型环境污染业务库。
`tooling/configs/gateway*.yaml` 现在以 `trace.mode` 作为权威运行模式，默认值为 `database`；仅当 `trace.mode=blockchain` 时才要求配置 `chain.rpc_url`、`chain.chain_id`、`chain.contract_address` 与 `chain.private_key`。
`tooling/configs/gateway*.yaml` 现在以 `auth.mode` 作为认证开关，默认值为 `disabled`；启用 OIDC 时需配置 `auth.oidc.issuer_url`、`auth.oidc.audience`、`auth.oidc.web_client_id`，以及 `auth.web.public_base_url`、`auth.web.app_base_url`、`auth.web.cookie_name`、`auth.web.cookie_secure`、`auth.web.cookie_same_site`。如果是首次在空库上启用 OIDC，还必须提供 `auth.bootstrap_admin_email`，用于预置首个本地管理员账号并在首次登录时绑定 OIDC `sub`。首次绑定预创建用户时，网关把 `email in access_token` 作为硬要求：Bearer Token 必须自带 `email` claim，系统不会额外调用 `userinfo`。上述字段也可通过 `LYCHEE_AUTH_MODE`、`LYCHEE_AUTH_OIDC_ISSUER_URL`、`LYCHEE_AUTH_OIDC_AUDIENCE`、`LYCHEE_AUTH_OIDC_WEB_CLIENT_ID`、`LYCHEE_AUTH_BOOTSTRAP_ADMIN_EMAIL`、`LYCHEE_AUTH_WEB_PUBLIC_BASE_URL`、`LYCHEE_AUTH_WEB_APP_BASE_URL`、`LYCHEE_AUTH_WEB_COOKIE_NAME`、`LYCHEE_AUTH_WEB_COOKIE_SECURE`、`LYCHEE_AUTH_WEB_COOKIE_SAME_SITE` 覆盖。Web Cookie 模式下，`cors.allowed_origins` 不能使用 `*`，并且需要把 `cors.allow_credentials` 设为 `true`。默认 `cookie_same_site=lax` 只支持 same-site 部署；如果前端站点与 Gateway 是 cross-site，则必须改成 `cookie_same_site=none` 且同时启用 `cookie_secure=true`。另外，所有基于 Cookie 的不安全请求与 WebSocket 握手都会校验 `Origin`，只有命中 `cors.allowed_origins` 的浏览器来源才会被接受。
Docker Compose 使用仓库自带的 `tooling/configs/gateway.compose.yaml`，其中容器内上游地址为 `http://inference-api:8000`。

前端运行时认证配置：

- `NUXT_PUBLIC_AUTH_MODE=disabled|oidc`
- `NUXT_PUBLIC_OIDC_TAURI_CLIENT_ID`
- `NUXT_PUBLIC_OIDC_SCOPE`

其中 Web 端不再直接持有 OIDC token，也不再需要公开的 `issuer/client_id/redirect_uri` 配置；这些信息都由 Gateway 在 `/v1/auth/login`、`/v1/auth/callback`、`/v1/auth/logout` 上托管。

本地开发默认建议：

```sh
LYCHEE_AUTH_MODE=disabled
NUXT_PUBLIC_AUTH_MODE=disabled
LYCHEE_SEED_DEFAULT_RESOURCES_ENABLED=true
```

如果需要在本地联调 OIDC 首次启用流程，至少补充：

```sh
LYCHEE_AUTH_MODE=oidc
LYCHEE_AUTH_BOOTSTRAP_ADMIN_EMAIL=admin@example.com
LYCHEE_AUTH_OIDC_WEB_CLIENT_ID=orchard-console-web
LYCHEE_AUTH_WEB_PUBLIC_BASE_URL=http://127.0.0.1:9000
LYCHEE_AUTH_WEB_APP_BASE_URL=http://127.0.0.1:3000
LYCHEE_AUTH_WEB_COOKIE_SAME_SITE=lax
```

同时确认你的 IdP 会把 `email` 放进发给网关的 `access_token`。以 Keycloak 为例，需要检查对应 client scope / protocol mapper 的 `email` claim 已启用，并且 `Add to access token` 为开启状态；否则首次登录绑定预创建用户会被拒绝。

## Docker

根目录 `docker-compose.yml` 使用新的职责分组路径：

- Inference API Dockerfile：`tooling/docker/Dockerfile.inference-api`
- Gateway Dockerfile：`tooling/docker/Dockerfile.gateway`

启动：

```sh
docker compose up --build
```

Compose 默认会让 gateway 读取 `tooling/configs/gateway.compose.yaml`；本地 `go run`、`bun run dev:gateway` 与 `tooling/scripts/stack.*` 继续使用 `tooling/configs/gateway.yaml`。

## Notes

- 前端默认必须经由 Gateway 访问 Inference API，不应直连 `services/inference-api`
- 共享契约新增字段时，优先更新 `shared/contracts` 后再同步到各服务
- `shared/python/lychee_common` 仅承载跨训练与推理共用的 Python 工具
