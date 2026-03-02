package mw

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ANSI colors for HTTP log line
const (
	_reset  = "\033[0m"
	_green  = "\033[32m"
	_yellow = "\033[33m"
	_red    = "\033[31m"
	_cyan   = "\033[36m"
	_dim    = "\033[2m"
	_bold   = "\033[1m"
)

func RequestLogger(logger *zap.Logger, isLocal bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if isLocal {
			statusColor := _green
			if status >= 500 {
				statusColor = _red
			} else if status >= 400 {
				statusColor = _yellow
			} else if status >= 300 {
				statusColor = _cyan
			}
			latencyStr := latency.String()
			if latency < time.Millisecond {
				latencyStr = fmt.Sprintf("%dµs", latency.Microseconds())
			} else {
				latencyStr = fmt.Sprintf("%.2fms", float64(latency.Nanoseconds())/1e6)
			}
			pathShow := path
			if len(pathShow) > 44 {
				pathShow = "..." + pathShow[len(pathShow)-41:]
			}
			fmt.Fprintf(os.Stdout, "  %s%-7s%s %s%-44s%s %s%3d%s %s%8s%s\n",
				_bold, method, _reset,
				_dim, pathShow, _reset,
				statusColor, status, _reset,
				_dim, latencyStr, _reset,
			)
		}

		logger.Info("http request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
		)
	}
}


