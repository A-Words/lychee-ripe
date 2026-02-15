# AGENTS.md

## 1. 项目目标
- 项目名称：`lychee-ripe`
- 目标：荔枝目标检测与成熟度识别，并提供后端推理服务与前端可视化能力。
- 主要类别映射（4 类）：`0=green`，`1=half`，`2=red`，`3=young`。

## 2. 目录约定
- `app/`：FastAPI 推理服务与 API（生产推理主路径）。
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

## 4. 常用命令
- 安装依赖：`uv sync`
- 启动服务：`uv run uvicorn app.main:app --reload`
- 运行测试：`uv run pytest -q`
- 训练模型：`uv run python training/train.py --data path/to/data.yaml --model yolo26n.pt`
- 评估模型：`uv run python training/eval.py --model artifacts/models/<exp>/weights/best.pt --data path/to/data.yaml`

## 5. 开发与改动规则
- 优先最小改动：只改与任务直接相关的文件。
- 不要擅自重命名顶层目录（如 `app/`、`training/`、`tests/`）。
- 修改配置或路径时，同步检查：
  - `README.md` 示例命令
  - `configs/*.yaml.example`
  - 相关测试
- 新增共享字段时，优先更新 `shared/` 下定义并保持前后端一致。
- 任何会影响行为的改动，至少运行一次相关测试。

## 6. 数据与版本信息
- 数据集引用：
  - Zhiqing, Zhao (2025), "lichi-maturity", Mendeley Data, V1, doi: `10.17632/c3rk9gv4w9.1`
- 本仓库不提交原始数据与大模型权重（遵循 `.gitignore`）。

## 7. 提交前检查清单
- 命令和路径示例是否可运行。
- 是否引入了新的硬编码绝对路径。
- 是否破坏 `configs/*.yaml.example` 的可用性。
- 是否保留类别映射一致性（green/half/red/young）。
- 测试是否通过，或已明确说明未执行原因。

## 8. 不确定时的默认策略
- 选择保守方案：优先保证现有 API 与训练流程兼容。
- 若需要结构性重构，先在 PR/变更说明里给出迁移步骤，再实施。
