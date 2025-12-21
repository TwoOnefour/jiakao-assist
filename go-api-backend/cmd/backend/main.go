package main

import (
	"github.com/gin-gonic/gin"
	"go-api-backend/internal/clients"
	"go-api-backend/internal/config"
	"go-api-backend/internal/services"
	"go-api-backend/internal/transport"
	"net/http"
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
	http.ListenAndServe(":8080", r)
}
