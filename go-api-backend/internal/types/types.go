package types

type SearchReq struct {
	Query string `json:"query" binding:"required"`
	TopK  int    `json:"topK" binding:"omitempty,min=1,max=50"`
}
type Hit struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Text     string                 `json:"text,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type SearchResp struct {
	Query string `json:"query"`
	TopK  int    `json:"topK"`
	Hits  []Hit  `json:"hits"`
}

type UserReq struct {
	Query string `json:"query"`
}
