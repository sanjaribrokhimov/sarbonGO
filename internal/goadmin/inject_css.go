package goadmin

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const injectHTMLBeforeHead = `<link rel="stylesheet" href="/admin-custom.css"><script src="/admin-custom.js"></script>`

// responseInjector перехватывает ответ и вставляет ссылку на наш CSS перед </head>.
type responseInjector struct {
	gin.ResponseWriter
	body        *bytes.Buffer
	code        int
	wroteHeader bool
}

func (r *responseInjector) WriteHeader(code int) {
	if !r.wroteHeader {
		r.code = code
		r.wroteHeader = true
	}
}

func (r *responseInjector) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

// InjectCSSMiddleware вставляет <link href="/admin-custom.css"> в HTML страниц админки.
// Вызывать ДО goadmin.Mount. Подходит только для запросов с префиксом /admin (кроме /admin-custom.css).
func InjectCSSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/admin") || path == "/admin-custom.css" || path == "/admin-custom.js" {
			c.Next()
			return
		}
		w := c.Writer
		buf := &bytes.Buffer{}
		ri := &responseInjector{ResponseWriter: w, body: buf, code: http.StatusOK}
		c.Writer = ri
		c.Next()
		if ri.body.Len() == 0 {
			return
		}
		out := ri.body.Bytes()
		if strings.Contains(ri.body.String(), "</head>") {
			html := strings.Replace(ri.body.String(), "</head>", injectHTMLBeforeHead+"\n</head>", 1)
			out = []byte(html)
		}
		if ri.wroteHeader {
			w.WriteHeader(ri.code)
		}
		w.Write(out)
	}
}

