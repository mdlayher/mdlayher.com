// Command mdlayher.com serves Matt Layher's personal website.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

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
	c := web.Content{
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

	handler := web.NewHandler(c)

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

		h := http.RedirectHandler(fmt.Sprintf("https://%s", c.Domain), http.StatusMovedPermanently)
		if err := http.ListenAndServe(":80", h); err != nil {
			log.Fatalf("failed to serve HTTP: %v", err)
		}
	}()

	log.Printf("starting HTTPS server for domain %q", c.Domain)

	domains := []string{
		c.Domain,
		// Also include www subdomain.
		"www." + c.Domain,
	}

	if err := http.Serve(autocert.NewListener(domains...), handler); err != nil {
		log.Fatalf("failed to serve HTTPS: %v", err)
	}
}
