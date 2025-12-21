package transport

import (
	"go-api-backend/internal/clients"
	"go-api-backend/internal/services"
	"go-api-backend/internal/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	rag services.RAGService
	llm clients.DeepSeek
}

func New(rag services.RAGService, llm clients.DeepSeek) *Handler {
	return &Handler{rag: rag, llm: llm}
}

// POST /search
func (h *Handler) SearchAndResponse(c *gin.Context) {
	var req types.SearchReq
	h.Cors(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, Err(&ErrorInfo{Code: "BAD_REQUEST", Message: err.Error()}))
		return
	}
	if req.TopK == 0 {
		req.TopK = 5
	}

	out, err := h.rag.SearchAndResponse(c, req.Query, req.TopK)
	
	if err != nil {
		c.JSON(http.StatusOK, Err(&ErrorInfo{Code: "SEARCH_FAILED", Message: err.Error(), Upstream: "worker/vectorize"}))
		return
	}

	c.JSON(http.StatusOK, OK(out))
}

func (h *Handler) Cors(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
	c.Writer.WriteHeaderNow()
}
