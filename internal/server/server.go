package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Launch(port int, indent int, char string, fn func(s string, indent int, char string) (string, error)) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html", []byte(getDocCache()))
	})

	router.POST("/format", func(c *gin.Context) {
		code := c.PostForm("code")
		updateDocCache(code, indent, char, fn)
		c.Redirect(http.StatusFound, "/")
	})

	router.GET("/base.css", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/css", CACHE_STYLESHEET)
	})

	router.GET("/base.js", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/javascript", CACHE_SCRIPT)
	})

	router.Run(fmt.Sprintf(":%d", port))
}
