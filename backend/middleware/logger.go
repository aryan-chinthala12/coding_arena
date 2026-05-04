package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs each request with security-relevant metadata.
// Useful for anomaly detection and incident forensics.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		userAgent := c.Request.UserAgent()

		/*
			Log at WARN level for client/server errors
			to make them easy to grep.
		*/
		if status >= 400 {
			log.Printf("[WARN] %s %s | %d | %v | ip=%s | ua=%s | errors=%s",
				method, path, status, latency, clientIP, userAgent, c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			log.Printf("[INFO] %s %s | %d | %v | ip=%s",
				method, path, status, latency, clientIP)
		}
	}
}
