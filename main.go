// Command mdlayher.com fetches and generates dynamic content for Matt Layher's
// static Hugo website.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mdlayher/mdlayher.com/internal/github"
	"github.com/mdlayher/mdlayher.com/internal/medium"
)

//go:generate go run main.go

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ghc := github.NewClient("mdlayher")
	repos, err := ghc.ListRepositories(ctx)
	if err != nil {
		log.Fatalf("failed to get GitHub repositories: %v", err)
	}

	if err := writeJSON("data/github.json", repos); err != nil {
		log.Fatalf("failed to create GitHub data file: %v", err)
	}

	mc := medium.NewClient("mdlayher")
	posts, err := mc.ListPosts(ctx)
	if err != nil {
		log.Fatalf("failed to get Medium posts: %v", err)
	}

	if err := writeJSON("data/medium.json", posts); err != nil {
		log.Fatalf("failed to create Medium data file: %v", err)
	}

	etag, ok, err := checkETag("https://raw.githubusercontent.com/mdlayher/talks/master/talks.json")
	if err != nil {
		log.Fatalf("failed to check talks ETag: %v", err)
	}

	if ok {
		// We have an ETag, write it to a metadata file so that changes are
		// picked up by the automated update script.
		if err := writeJSON("data/.talks-etag.json", etag); err != nil {
			log.Fatalf("failed to create talks ETag file: %v", err)
		}
	}
}

func writeJSON(file string, v interface{}) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("failed to create data file %q: %v", file, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")

	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("failed to JSON encode to data file %q: %v", file, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close data file %q: %v", file, err)
	}

	return nil
}

func checkETag(uri string) (string, bool, error) {
	c := &http.Client{
		Timeout: 5 * time.Second,
	}

	res, err := c.Head(uri)
	if err != nil {
		return "", false, fmt.Errorf("failed to send HTTP HEAD %q: %v", uri, err)
	}
	_ = res.Body.Close()

	// Ensure an ETag was sent.
	etag := strings.Trim(res.Header.Get("ETag"), `"`)
	ok := etag != ""

	return etag, ok, nil
}
