package main

import (
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/esote/openshim2"
)

func init() {
	if err := openshim2.LazySysctls(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	const (
		gen    = "gen/"
		static = "static/"

		port = ":8442"
	)

	if err := openshim2.Unveil(gen, "r"); err != nil {
		log.Fatal(err)
	}

	if err := openshim2.Unveil(static, "r"); err != nil {
		log.Fatal(err)
	}

	if err := openshim2.Pledge("stdio rpath inet", ""); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	patterns := []string{
		"/css/", "/dl/", "/img/", "/robots.txt",
	}

	staticHandler := &handler{static, func(w http.ResponseWriter) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		const tenYears = 10 * 365 * 24 * time.Hour
		expires := time.Now().Add(tenYears).UTC().Format(time.RFC1123)
		w.Header().Set("Expires", expires)
	}}

	for _, pattern := range patterns {
		mux.Handle(pattern, staticHandler)
	}

	mux.Handle("/", &handler{gen, func(w http.ResponseWriter) {
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; style-src 'self'; img-src 'self'")
	}})

	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

type handler struct {
	pattern string
	f       func(w http.ResponseWriter)
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000;"+
		"includeSubDomains;preload")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "deny")
	w.Header().Set("X-XSS-Protection", "1")

	h.f(w)

	http.ServeFile(w, r, filepath.Join(h.pattern, r.URL.Path[1:]))
}
