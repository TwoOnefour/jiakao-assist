package services

import (
	"context"
	"go-api-backend/internal/clients"
	"go-api-backend/internal/types"
	"go-api-backend/internal/util"
)

type RAGService interface {
	Search(ctx context.Context, query string, topK int) (*types.SearchResp, error)
	AnswerStream(ctx context.Context, query string, k int, f func(delta string), f2 func(stage string, msg string)) ([]types.Hit, error)
}

type ragService struct {
	worker    clients.Worker
	deepseek  clients.DeepSeek
	topK      int
	threshold float64
}

type RAGDeps struct {
	Worker    clients.Worker
	TopK      int
	Threshold float64
	Deepseek  clients.DeepSeek
}

func NewRAG(d RAGDeps) RAGService {
	if d.TopK <= 0 {
		d.TopK = 5
	}
	if d.Threshold <= 0 {
		d.Threshold = 0.35
	}
	return &ragService{worker: d.Worker, topK: d.TopK, threshold: d.Threshold, deepseek: d.Deepseek}
}

func (r *ragService) Search(ctx context.Context, query string, topK int) (*types.SearchResp, error) {
	if topK <= 0 {
		topK = r.topK
	}
	out, err := r.worker.Search(ctx, query, topK)
	if err != nil {
		return nil, err
	}
	// 这里可做命中过滤/排序/阈值裁剪等业务逻辑
	return out, nil
}

func (r *ragService) AnswerStream(
	ctx context.Context,
	query string, topK int,
	onDelta func(string),
	onStatus func(stage, msg string),
) ([]types.Hit, error) {

	if onStatus != nil {
		onStatus("search", "query vector index")
	}
	// 1) 调 Worker /search -> hits
	out, err := r.worker.Search(ctx, query, topK)
	if err != nil {
		return nil, err
	}

	// 2) 命中判断 -> 选择 distill 或 v3
	mode := "llm"
	if len(out.Hits) == 0 || out.Hits[0].Score < r.threshold {
		mode = "fallback"
	}
	if onStatus != nil {
		if mode == "llm" {
			onStatus("llm", "deepseek")
		} else {
			onStatus("fallback", "deepseek-v3")
		}
	}

	// 3) 组装 prompt → deepseek 流式
	promptStr := util.BuildPrompt(query, out.Hits)
	var messages []clients.ChatMessage
	messages = append(messages, clients.ChatMessage{
		Role:    "system",
		Content: "系统：你是驾考助手。仅依据【资料】回答；无法依据时说不知道。要求：\n1) 先给结论；2) 在答案末尾标注引用编号（如 [1][3]）；3) 不要编造资料中不存在的内容。",
	})
	messages = append(messages, clients.ChatMessage{
		Role:    "user",
		Content: promptStr,
	})

	prompt := &clients.Prompt{
		Messages: messages,
	}
	_, err = r.deepseek.Stream(ctx, r.deepseek.GetName(), prompt.Messages, prompt.Options, onDelta)
	if err != nil {
		return nil, err
	}

	return out.Hits, nil
}
