package api

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

// SPEC-129: la tabla MIME de Go (y la imagen distroless) no incluye
// .webmanifest. Sin este registro el manifest se serviría como text/plain y
// Chrome lo rechazaría.
func init() {
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

const (
	pwaManifestCacheControl      = "no-cache"
	pwaServiceWorkerCacheControl = "no-cache"
	pwaIconsCacheControl         = "public, max-age=604800"
	pwaAssetsCacheControl        = "public, max-age=31536000, immutable"
)

// pwaManifestHandler sirve el manifest con MIME y caché correctos.
func pwaManifestHandler(staticDir string) http.HandlerFunc {
	return staticFileHandler(staticDir, "manifest.webmanifest", "application/manifest+json", pwaManifestCacheControl)
}

// pwaServiceWorkerHandler sirve el service worker en el scope raíz.
func pwaServiceWorkerHandler(staticDir string) http.HandlerFunc {
	base := staticFileHandler(staticDir, "sw.js", "text/javascript; charset=utf-8", pwaServiceWorkerCacheControl)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Service-Worker-Allowed", "/")
		base(w, r)
	}
}

// staticFileHandler sirve un archivo puntual de public/ y devuelve 404 si no
// existe, evitando que el fallback SPA lo reemplace por index.html.
func staticFileHandler(staticDir, name, contentType, cacheControl string) http.HandlerFunc {
	file := filepath.Join(staticDir, name)
	return func(w http.ResponseWriter, r *http.Request) {
		if info, err := os.Stat(file); err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", cacheControl)
		http.ServeFile(w, r, file)
	}
}

// withCacheControl aplica un Cache-Control a un handler existente.
func withCacheControl(cacheControl string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheControl)
		next.ServeHTTP(w, r)
	})
}
