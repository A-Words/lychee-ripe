# lychee-ripe

荔枝目标检测与成熟度识别项目，提供：

- FastAPI 推理服务（`app/`）
- Go 网关服务（`gateway/`）
- Nuxt 前端可视化（`frontend/`，支持 Web/Desktop）
- 训练与评估脚本（`training/`）

成熟度类别映射（4 类）：

- `0 = green`
- `1 = half`
- `2 = red`
- `3 = young`

共享常量来源：`shared/constants/ripeness.json`  
接口契约来源：`shared/schemas/openapi.yaml`

---

## 1. 系统架构与目录

调用链（默认）：

`Web/Desktop Frontend -> Go Gateway -> FastAPI Inference`

关键目录：

- `app/`：推理服务与 API
- `gateway/`：对外 API、鉴权、限流、观测
- `frontend/`：前端可视化客户端（Nuxt + Tauri）
- `training/`：训练与评估脚本
- `tests/`：Python/Go/前端测试
- `shared/`：共享常量与 OpenAPI 契约
- `configs/`：配置模板与本地配置
- `scripts/`：联调、训练、评估、校验脚本
- `artifacts/`：模型、指标、日志产物

关键路径约定：

- 训练输出：`artifacts/models/`
- 评估输出：`artifacts/metrics/`
- 推理模型配置：`configs/model.yaml`
- 网关配置：`configs/gateway.yaml`
- OpenAPI 契约：`shared/schemas/openapi.yaml`

---

## 2. 环境要求

- Python `>= 3.11`（见 `pyproject.toml`）
- Go（见 `gateway/go.mod`，当前为 `go 1.25.6`）
- Bun（前端）
- 可选：NVIDIA GPU + `nvidia-smi`（用于自动选择 `uv.lock`）

推荐依赖流程（先锁文件再安装）：

### Linux/macOS (sh)

```bash
sh scripts/switch-lock.sh --target auto
uv sync
bun install --cwd frontend
```

### Windows (PowerShell)

```powershell
powershell -ExecutionPolicy Bypass -File scripts/switch-lock.ps1 -Target auto
uv sync
bun install --cwd frontend
```

---

## 3. 快速开始

### 3.1 准备配置文件

先从模板复制本地配置（本地 `.yaml` 不提交）：

### Linux/macOS (sh)

```bash
cp configs/model.yaml.example configs/model.yaml
cp configs/service.yaml.example configs/service.yaml
cp configs/gateway.yaml.example configs/gateway.yaml
```

### Windows (PowerShell)

```powershell
Copy-Item configs/model.yaml.example configs/model.yaml
Copy-Item configs/service.yaml.example configs/service.yaml
Copy-Item configs/gateway.yaml.example configs/gateway.yaml
```

### 3.2 一键联调启动（推荐）

### Linux/macOS (sh)

```bash
sh scripts/stack.sh --app-host 127.0.0.1 --app-port 8000 --gateway-config configs/gateway.yaml --frontend-host 127.0.0.1 --frontend-port 3000
```

### Windows (PowerShell)

```powershell
powershell -ExecutionPolicy Bypass -File scripts/stack.ps1 -AppHost 127.0.0.1 -AppPort 8000 -GatewayConfig configs/gateway.yaml -FrontendHost 127.0.0.1 -FrontendPort 3000
```

默认端口：

- app：`8000`
- gateway：`9000`
- frontend：`3000`

### 3.3 分服务启动

#### app（FastAPI）

```bash
sh scripts/app.sh --host 127.0.0.1 --port 8000
```

```powershell
powershell -ExecutionPolicy Bypass -File scripts/app.ps1 -Host 127.0.0.1 -Port 8000
```

#### gateway（Go）

```bash
sh scripts/gateway.sh --config configs/gateway.yaml
```

```powershell
powershell -ExecutionPolicy Bypass -File scripts/gateway.ps1 -Config configs/gateway.yaml
```

#### frontend（Web）

```bash
sh scripts/frontend.sh --host 127.0.0.1 --port 3000
```

```powershell
powershell -ExecutionPolicy Bypass -File scripts/frontend.ps1 -Host 127.0.0.1 -Port 3000
```

#### frontend（Desktop / Tauri）

```bash
sh scripts/desktop.sh
```

```powershell
powershell -ExecutionPolicy Bypass -File scripts/desktop.ps1
```

---

## 4. 配置说明

### `configs/model.yaml`

- `model_path`：在线推理模型路径（为空时使用默认模型加载行为）
- `conf_threshold`：检测置信度阈值
- `nms_iou`：NMS IoU 阈值
- `device`：`auto` / `cpu` / `cuda`（或 CUDA 设备编号）

### `configs/service.yaml`

- `app_name`：服务名
- `schema_version`：响应协议版本
- `max_upload_mb`：单图接口上传大小限制（MB）

### `configs/gateway.yaml`

- `server`：网关监听地址与读写超时
- `upstream.base_url`：上游 FastAPI 地址（默认 `http://127.0.0.1:8000`）
- `db.driver`：数据库驱动（`sqlite` 或 `postgres`）
- `db.dsn`：连接串（sqlite 文件路径或 postgres DSN）
- `db.max_open_conns` / `db.max_idle_conns` / `db.conn_max_lifetime_s`：连接池参数
- `db.sqlite`：SQLite 参数（`journal_mode`、`busy_timeout_ms`）
- `db.postgres`：PostgreSQL 参数（`ssl_mode`、`schema`）
- `chain`：EVM 锚定参数（`enabled`、`rpc_url`、`chain_id`、`contract_address`、`private_key`、`tx_timeout_s`、`receipt_poll_interval_ms`）
- `auth`：API Key 开关与密钥列表
- `rate_limit`：限流参数
- `cors`：跨域策略
- `logging`：日志级别与格式

PostgreSQL 配置示例：

```yaml
db:
  driver: "postgres"
  dsn: "postgres://postgres:postgres@127.0.0.1:5432/lychee_ripe?sslmode=disable"
  max_open_conns: 10
  max_idle_conns: 5
  conn_max_lifetime_s: 300
  postgres:
    ssl_mode: "disable"
    schema: "public"
```

EVM 链配置示例（本地测试链）：

```yaml
chain:
  enabled: true
  rpc_url: "http://127.0.0.1:8545"
  chain_id: "31337"
  contract_address: "0x1234567890abcdef1234567890abcdef12345678"
  private_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  tx_timeout_s: 30
  receipt_poll_interval_ms: 500
```

说明：`private_key` 仅用于本地开发测试链。生产环境应替换为专用密钥管理方案，避免明文配置。

### 环境变量入口

- `LYCHEE_MODEL_CONFIG`（app 模型配置路径）
- `LYCHEE_SERVICE_CONFIG`（app 服务配置路径）
- `LYCHEE_GATEWAY_CONFIG`（gateway 配置路径）

前端网关地址默认在 `frontend/nuxt.config.ts` 中为 `http://127.0.0.1:9000`，可通过 Nuxt 公共运行时配置覆盖（例如 `NUXT_PUBLIC_GATEWAY_BASE`）。

公众溯源查询页路由：

- 手动输入页：`/trace`
- 二维码落地页：`/trace/{trace_code}`

---

## 5. 训练与评估

### 5.1 训练

```bash
sh scripts/train.sh --data data/lichi/data.yaml --name lychee_v1
```

```powershell
powershell -ExecutionPolicy Bypass -File scripts/train.ps1 -Data data/lichi/data.yaml -Name lychee_v1
```

可选导出 ONNX：

```bash
sh scripts/train.sh --data data/lichi/data.yaml --name lychee_v1 --export-onnx
```

### 5.2 评估

```bash
sh scripts/eval.sh --data data/lichi/data.yaml --exp lychee_v1
```

```powershell
powershell -ExecutionPolicy Bypass -File scripts/eval.ps1 -Data data/lichi/data.yaml -Exp lychee_v1
```

默认产物：

- checkpoint：`artifacts/models/<exp>/weights/best.pt`
- 指标：`artifacts/metrics/<exp>-eval_metrics.json`

---

## 6. 质量检查与提交流程

一键检查：

```bash
sh scripts/verify.sh
```

```powershell
powershell -ExecutionPolicy Bypass -File scripts/verify.ps1
```

分层检查：

```bash
uv run pytest -q
go test ./gateway/...
bun run --cwd frontend typecheck
bun run --cwd frontend test
bun run --cwd frontend generate
```

提交前检查清单（精简版）：

- 命令示例是否可运行，路径是否正确
- 未引入硬编码绝对路径
- `configs/*.yaml.example` 可用性不被破坏
- 类别映射保持 `green/half/red/young` 一致
- `shared/schemas/openapi.yaml` 与 `app/`、`gateway/`、`frontend/` 字段一致
- 前端调用路径保持 `frontend -> gateway -> app`
- 前端颜色与标签映射与 `shared/constants/ripeness.json` 一致
- 摄像头切换能力可用（空闲/识别中切换、拔插刷新、上次选择恢复）

---

## 7. 已知限制

- 当前 README 不提供 Docker 作为可执行主流程。
- 原因：`docker/Dockerfile` 依赖的 `requirements.txt` 当前未在仓库中提供，按现状构建可能失败。
- 待容器构建链路修复后，再补充 Docker 章节。

---

## 8. 常见问题（FAQ）

### Q1：为什么前端不能直连 `app/`？

项目约定前端只调用 `gateway/`。网关负责统一鉴权、限流、日志与跨域策略，避免把这些能力散落在前端或推理服务中。

### Q2：为什么 `/v1/health` 可能返回 `degraded`？

服务启动时如果模型加载/预热失败，FastAPI 会保持进程可用并暴露健康信息。此时可见 `status=degraded`，便于观测与排障。

### Q3：摄像头拔插后前端如何处理？

前端会监听设备变化并刷新列表，支持空闲态与识别中切换，并保留上次选择设备。当前摄像头不可用时会尝试回退到可用设备。

---

## 9. 参考与数据

- 数据集引用：Zhiqing, Zhao (2025), "lichi-maturity", Mendeley Data, V1, doi: `10.17632/c3rk9gv4w9.1`
- 本仓库不提交原始数据与大模型权重（遵循 `.gitignore`）
