# 已完成 ✅

* [x] 本地 **RAG + 向量** 流水线跑通（BGE small zh → Chroma/Vectorize）
* [x] 向量维度/metric 策略确定（512 + cosine，归一化）
* [x] 小数据集准备（≈150 条）
* [x] **Cloudflare Vectorize 检索**最小接口打通（/query → 返回 top-k 文本+score）

# 进行中 ⏳（优先收口）

* [ ] **HF Worker /embed**（bge-small-zh-v1.5，mean-pool + L2）部署并可用

# 接下来 ▶️（核心闭环）

* [ ] **Go 网关 + WebSocket**：单通道推送阶段状态（embedding → search → llm → delta → done）
* [ ] **RAG 生成**：命中走 *deepseek-distill*（带引用）；未命中走 *deepseek-v3*（降级）
* [ ] **Prompt 模板化**：RAG 与 fallback 两套模板（变量：query、contexts、要求），版本号埋点
* [ ] **最小前端（Vue）**：输入框 + 进度条/状态泡泡 + top-k 列表 + 流式答案区

# 基础设施 🔜

* [ ] **Redis**：embedding 缓存、search 缓存、WS 会话状态（短 TTL）
* [ ] **数据库**（Postgres/MySQL）：documents/chunks、prompt_templates、requests/answers/citations、feedback
* [ ] **秘钥与配置**：HF/DeepSeek/CF Token 环境化，前端不暴露

# 监控面板 admin-metrics 🔜

* [ ] 命中率/未命中率（按天/小时）
* [ ] Top-K 分布 & 相似度直方图（阈值可视）
* [ ] 延迟分解（embed/search/llm p50/p90）
* [ ] 缓存命中率（embedding/search）
* [ ] 上游错误码统计（HF/DeepSeek/Vectorize）

# 质量与策略 🌟（可选加分）

* [ ] 召回率@k / 精准率@k 的小型标注集评测
* [ ] 幻觉率抽样评审 & 引用可用性统计
* [ ] 阈值自动建议（基于相似度分布）
* [ ] 请求回放（查看当时 top-k、prompt、答案、引用）

# 工程卫生 🌟

* [ ] 429/5xx 指数退避与重试；速率限制（Redis）
* [ ] 结构化日志与追踪（请求ID、tokens、模型、路径 rag/v3）
* [ ] 单元/集成小测（embed、query、fallback 路径）
* [ ] Demo 脚本与样例数据（离线演示不依赖外网）
