## todolist
* [ ] 定义 `types.go`（Hit/WSMsg/Errors）与 `clients`/`services` 接口
* [ ] 写 `handlers/ws_answer.go`、`handlers/search.go`，只调 `RAG.Answer`
* [ ] `services/rag.go`：命中阈值、路由 distill/v3、回调 onDelta
* [ ] `clients` 假实现跑通 → 再替换为真实 Worker/DeepSeek
* [ ] 接入 Redis（search/ans 缓存）→ 接入 DB（requests/answers/citations）
* [ ] 埋点（latency、命中率、错误码）
