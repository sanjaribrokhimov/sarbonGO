// Раздача Swagger UI и OpenAPI из встроенных файлов (embed).
package docs

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed swagger/*
var swaggerFS embed.FS

// SwaggerFS — встроенная папка swagger для HTTP file server (без префикса "swagger" в путях).
var SwaggerFS http.FileSystem = mustSub()

func mustSub() http.FileSystem {
	sub, err := fs.Sub(swaggerFS, "swagger")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}

// SwaggerIndex отдаёт /swagger/index.html (главная страница Swagger UI).
func SwaggerIndex(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = "/index.html"
	http.FileServer(SwaggerFS).ServeHTTP(w, r)
}
