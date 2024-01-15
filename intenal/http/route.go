package http

import (
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/sites", listHandler)
	r.POST("/sites", createHandler)
	return r
}
