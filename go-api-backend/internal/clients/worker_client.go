package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go-api-backend/internal/types"
)

type Worker interface {
	Search(ctx context.Context, query string, topK int) (*types.SearchResp, error)
}

type workerClient struct {
	base *url.URL
	http *http.Client
	// token string // 如需鉴权可加
}

func NewWorker(baseURL string) (Worker, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	return &workerClient{
		base: u,
		http: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (w *workerClient) Search(ctx context.Context, query string, topK int) (*types.SearchResp, error) {
	payload := types.SearchReq{Query: query, TopK: topK}
	bs, _ := json.Marshal(payload)

	u := *w.base
	u.Path = "/search"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(bs))
	if err != nil {
		return nil, fmt.Errorf("new req: %w", err)
	}
	req.Header.Set("User-Agent", "rag-gateway")
	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("Authorization","Bearer "+w.token)

	resp, err := w.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("worker /search status=%d resp=%s", resp.StatusCode, string(b))
	}

	// 包壳
	var wrapped struct {
		Code    int `json:"code"`
		Message struct {
			Data *types.SearchResp `json:"data"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapped); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if wrapped.Code != 0 { // 你的 CommonResp 规范建议 0=成功
		return nil, fmt.Errorf("worker error code=%d", wrapped.Code)
	}
	return wrapped.Message.Data, nil
}
