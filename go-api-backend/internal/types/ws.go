package types

type WSAsk struct {
	Type  string `json:"type"` // "ask"
	Query string `json:"query" binding:"required"`
	TopK  int    `json:"top_k"`
}

type WSStatus struct {
	Type  string `json:"type"`  // "status"
	Stage string `json:"stage"` // "embedding" | "search" | "llm" | "fallback"
	Msg   string `json:"msg"`
}

type WSHits struct {
	Type  string `json:"type"` // "hits"
	Items []Hit  `json:"items"`
}

type WSError struct {
	Type string `json:"type"` // "error"
	Msg  string `json:"msg"`
}

type WSDone struct {
	Type string `json:"type"` // "done"
}

type WSPing struct {
	Type string `json:"type"`
}
