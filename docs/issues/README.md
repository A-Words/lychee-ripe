# GitHub Issue Backlog（AI Coding）

## 使用方式
1. 按 `I01 -> I13` 依赖顺序在 GitHub 创建 issue。
2. 每个 issue 文件可直接复制到 GitHub issue 正文。
3. 标题建议格式：`[P0][gateway] I04 创建批次接口`。

## 依赖主干
`I01 -> I02 -> I03 -> I04/I05/I06/I07/I08/I09 -> I10/I11/I12 -> I13`

## 并行规则
1. `I04~I09` 可在 `I01~I03` 完成后并行。
2. `I10~I12` 可在 `I01 + I04 + I06 + I08` 完成后并行。
3. `I13` 最后执行，做统一回归与文档收口。

## 固定约束
1. 保持现有接口不变：`/v1/infer/image`、`/v1/infer/stream`、`/v1/health`。
2. 新增接口：`POST /v1/batches`、`GET /v1/batches/{batch_id}`、`GET /v1/trace/{trace_code}`、`GET /v1/dashboard/overview`、`POST /v1/batches/reconcile`。
3. 未成熟果规则：
- `unripe_count = green + young`
- `unripe_ratio = unripe_count / total`
- 阈值默认 `0.15`
- `unripe_handling` 默认 `sorted_out`
- 当 `unripe_ratio > 0.15` 且未二次确认时，禁止创建批次
4. 全部 issue 使用中文，仅标注优先级，不做 SP。

## 全局验收场景
1. 建批上链成功：返回 `trace_code` 且 `verify_status=pass`。
2. 链故障降级：建批成功但 `pending_anchor`，补链后转 `anchored`。
3. 未成熟果拦截：`unripe_ratio > 0.15` 时必须二次确认。
4. 安全策略：写接口无 key 拒绝，查询接口公开。
5. 数据一致性：篡改摘要后查询返回 `fail`。
6. 回归：识别接口与前端识别链路不退化。
