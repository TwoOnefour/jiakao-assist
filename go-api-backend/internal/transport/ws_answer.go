package transport

import (
	"go-api-backend/internal/services"
	"net/http"
	"time"
	
	"go-api-backend/internal/types"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	rag services.RAGService
}

func NewWS(r services.RAGService) *WSHandler { return &WSHandler{rag: r} }

// 允许跨域（按需收紧）
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// GET /ws/answer
func (h *WSHandler) Answer(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 心跳：服务端 ping 维持长连
	pingTicker := time.NewTicker(20 * time.Second)
	defer pingTicker.Stop()
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 读第一条 ask
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	var ask types.WSAsk
	if err := conn.ReadJSON(&ask); err != nil {
		_ = conn.WriteJSON(types.WSError{Type: "error", Msg: "invalid ask"})
		return
	}
	if ask.TopK <= 0 {
		ask.TopK = 5
	}

	send := func(v any) {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_ = conn.WriteJSON(v)
	}

	// 发送状态：embedding
	send(types.WSStatus{Type: "status", Stage: "embedding", Msg: "embed query (HF)"})

	// 在 RAG 服务里做编排：embedding → search → model route
	// AnswerStream 需要你在 services 实现：返回命中的 hits，并且在生成时按回调逐段推送
	hits, err := h.rag.AnswerStream(c, ask.Query, ask.TopK, func(delta string) {
		send(map[string]string{"type": "delta", "text": delta})
	}, func(stage, msg string) {
		// 可选：服务层回调状态（如 "search"、"llm"/"fallback"）
		send(types.WSStatus{Type: "status", Stage: stage, Msg: msg})
	})
	if err != nil {
		send(types.WSError{Type: "error", Msg: err.Error()})
		return
	}

	// 把 top-k 命中先回给前端
	send(types.WSHits{Type: "hits", Items: hits})

	// 心跳 loop（直到服务层完成并发出 done）
	done := make(chan struct{})
	go func() {
		// 这里不再读更多客户端消息；如需支持取消，可在此读 "cancel"
		for range pingTicker.C {
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				close(done)
				return
			}
		}
	}()

	// 服务层生成完毕后，你应当从 AnswerStream 里发送最终 done，这里补发兜底
	send(types.WSDone{Type: "done"})
	close(done)
}
