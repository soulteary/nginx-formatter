package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func Launch(port int) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		response := struct {
			Message string `json:"message"`
		}{
			Message: "今天天气真好呀",
		}

		c.JSON(200, response)
	})

	router.Run(fmt.Sprintf(":%d", port))
}
