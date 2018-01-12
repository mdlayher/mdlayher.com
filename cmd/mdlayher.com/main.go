// Command mdlayher.com serves Matt Layher's personal website.
package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os/user"
	"path/filepath"
	"time"

	"github.com/mdlayher/mdlayher.com/internal/github"
	"github.com/mdlayher/mdlayher.com/internal/httptalks"
	"github.com/mdlayher/mdlayher.com/internal/medium"
	"github.com/mdlayher/mdlayher.com/internal/web"

	"golang.org/x/crypto/acme/autocert"
)

func main() {
	var (
		useTLS = flag.Bool("tls", false, "use TLS with Let's Encrypt (production mode)")
	)

	flag.Parse()

	// Information is hard-coded for simplicity of deployment, but this could
	// be easily changed in the future.
	static := web.StaticContent{
		Domain:  "mdlayher.com",
		Name:    "Matt Layher",
		Tagline: "Software Engineer. Go, Linux, and open source software enthusiast. On and ever upward.",
		Links: []web.Link{
			{
				Title: "GitHub",
				Link:  "https://github.com/mdlayher",
			},
			{
				Title: "Medium (blog)",
				Link:  "https://medium.com/@mdlayher",
			},
			{
				Title: "Twitter",
				Link:  "https://twitter.com/mdlayher",
			},
		},
	}

	// Retrieve external metadata for display, cache for set amount of time.
	ghc := github.NewClient("mdlayher", 12*time.Hour)
	mc := medium.NewClient("mdlayher", 24*time.Hour)
	htc := httptalks.NewClient("https://raw.githubusercontent.com/mdlayher/talks/master/talks.json", 24*time.Hour)

	handler := web.NewHandler(static, ghc, mc, htc)

	// Enable development mode when not using TLS.
	if !*useTLS {
		log.Println("starting HTTP development server")

		if err := http.ListenAndServe(":8080", handler); err != nil {
			log.Fatalf("failed to serve development HTTP: %v", err)
		}
	}

	user, err := user.Current()
	if err != nil {
		log.Fatalf("failed to get current user: %v", err)
	}

	// Same location as autocert.NewListener uses.
	dir := filepath.Join(user.HomeDir, ".cache", "golang-autocert")

	// Use Let's Encrypt for TLS.
	m := &autocert.Manager{
		Cache:  autocert.DirCache(dir),
		Prompt: autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(
			static.Domain,
			// Also include www subdomain.
			"www."+static.Domain,
		),
	}

	// Always redirect HTTP to HTTPS, and provide the handler necessary
	// for Let's Encrypt to perform the http-01 challenge.
	go func() {
		log.Println("starting HTTP redirect server")

		if err := http.ListenAndServe(":http", m.HTTPHandler(nil)); err != nil {
			log.Fatalf("failed to serve HTTP: %v", err)
		}
	}()

	log.Printf("starting HTTPS server for domain %q", static.Domain)

	s := &http.Server{
		Addr:      ":https",
		Handler:   handler,
		TLSConfig: &tls.Config{GetCertificate: m.GetCertificate},
	}

	if err := s.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("failed to serve HTTPS: %v", err)
	}
}
