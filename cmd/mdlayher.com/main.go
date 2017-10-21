// Command mdlayher.com serves Matt Layher's personal website.
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/mdlayher/mdlayher.com/internal/github"
	"github.com/mdlayher/mdlayher.com/internal/gittalks"
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
	gtc := gittalks.NewClient("https://github.com/mdlayher/talks", 24*time.Hour)

	handler := web.NewHandler(static, ghc, mc, gtc)

	// Enable development mode when not using TLS.
	if !*useTLS {
		log.Println("starting HTTP development server")

		if err := http.ListenAndServe(":8080", handler); err != nil {
			log.Fatalf("failed to serve development HTTP: %v", err)
		}
	}

	// Always redirect HTTP to HTTPS.
	go func() {
		log.Println("starting HTTP redirect server")

		redirect := web.NewRedirectHandler(static.Domain)
		if err := http.ListenAndServe(":80", redirect); err != nil {
			log.Fatalf("failed to serve HTTP: %v", err)
		}
	}()

	log.Printf("starting HTTPS server for domain %q", static.Domain)

	domains := []string{
		static.Domain,
		// Also include www subdomain.
		"www." + static.Domain,
	}

	if err := http.Serve(autocert.NewListener(domains...), handler); err != nil {
		log.Fatalf("failed to serve HTTPS: %v", err)
	}
}
