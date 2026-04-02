# Lychee Ripe

荔枝目标检测与成熟度识别 Monorepo，按职责分组为 `clients/`、`services/`、`shared/`、`mlops/`、`tooling/`、`tests/`。

## Overview

- `clients/orchard-console`：Nuxt Web + Tauri Desktop，果园业务操作前端
- `services/inference-api`：FastAPI 推理服务
- `services/gateway`：Go 网关，负责鉴权、限流、日志、代理与 WebSocket 透传
- `shared/contracts`：共享常量与 OpenAPI 契约
- `shared/python`：训练与推理共用的 Python helper
- `mlops/training`：训练与评估脚本
- `mlops/data`：数据集与标注
- `mlops/artifacts`：模型、指标、日志与网关本地数据库
- `tooling/configs`：配置模板与本地配置
- `tooling/scripts`：启动、训练、评估、校验脚本
- `tooling/docker`：镜像构建文件
- `tests/stack`：跨服务 smoke 测试

调用链路固定为 `frontend -> gateway -> api`。

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

分服务直接启动：

```sh
cd services/inference-api && uv run --extra cpu python -m uvicorn app.main:app --reload --host 127.0.0.1 --port 8000
go run ./services/gateway/cmd/gateway --config tooling/configs/gateway.yaml
cd clients/orchard-console && bun run dev -- --host 127.0.0.1 --port 3000
cd clients/orchard-console && bun run tauri:dev
```

脚本入口：

```sh
sh tooling/scripts/inference-api.sh --target cpu --host 127.0.0.1 --port 8000
sh tooling/scripts/gateway.sh --config tooling/configs/gateway.yaml
sh tooling/scripts/orchard-console.sh --host 127.0.0.1 --port 3000
sh tooling/scripts/desktop.sh
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
bun run --filter @lychee-ripe/training train -- --data mlops/data/lichi/data.yaml --name custom_run
bun run --filter @lychee-ripe/training eval -- --model mlops/artifacts/models/custom_run/weights/best.pt --data mlops/data/lichi/data.yaml --output mlops/artifacts/metrics/custom_run.json
```

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
bun run verify
```

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

## Config

- 推理模型配置：`tooling/configs/model.yaml`
- 服务配置：`tooling/configs/service.yaml`
- 网关配置：`tooling/configs/gateway.yaml`

默认模型路径、网关默认配置与本地 sqlite 路径都已改成 repo-root 相对解析，直接从仓库根或各服务目录运行都可以。
`tooling/configs/gateway.yaml.example` 默认面向本地直启，`upstream.base_url` 使用 `http://127.0.0.1:8000`。
Docker Compose 使用仓库自带的 `tooling/configs/gateway.compose.yaml`，其中容器内上游地址为 `http://inference-api:8000`。

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
