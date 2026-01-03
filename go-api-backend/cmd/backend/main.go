package main

import (
	"go-api-backend/internal/clients"
	"go-api-backend/internal/config"
	"go-api-backend/internal/public"
	"go-api-backend/internal/services"
	"go-api-backend/internal/transport"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	conf, err := config.Load()
	if err != nil {
		panic(err)
	}

	worker, _ := clients.NewWorker(conf.Worker.BaseURL)
	deepseek, _ := clients.NewDeepSeek(conf.DeepSeek.BaseUrl, conf.DeepSeek.APIKey, conf.DeepSeek.Name, conf.DeepSeek.BasePath)
	rag := services.NewRAG(services.RAGDeps{Worker: worker, TopK: 5, Threshold: 0.35, Deepseek: deepseek})

	r := gin.New()
	r.Use(gin.Recovery())
	h := transport.New(rag, deepseek)
	//wsH := transport.NewWS(rag)
	//r.GET("/ws/answer", wsH.Answer)

	r.POST("/generate", h.SearchAndResponse)
	r.OPTIONS("/generate", h.Cors)
	staticFS := public.Dist
	fileServer := http.FileServer(http.FS(staticFS))
	r.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/generate") {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		c.JSON(404, gin.H{"code": 404, "msg": "Not Found"})
	})
	http.ListenAndServe(":"+conf.Server.Port, r)
}
