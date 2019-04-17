// Command mdlayher.com fetches and generates dynamic content for Matt Layher's
// static Hugo website.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mdlayher/mdlayher.com/internal/github"
	"github.com/mdlayher/mdlayher.com/internal/httptalks"
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

	// If mdlayher.com is present in the list, place it at the very end, to
	// avoid unneeded churn during the daily automatic update.
	var (
		repoIdx     = -1
		websiteRepo *github.Repository
	)

	for i, r := range repos {
		if r.Name == "mdlayher.com" {
			repoIdx = i
			websiteRepo = r
			break
		}
	}

	if repoIdx != -1 && websiteRepo != nil {
		// https://github.com/golang/go/wiki/SliceTricks#delete
		repos = append(repos[:repoIdx], repos[repoIdx+1:]...)
		repos = append(repos, websiteRepo)
	}

	if err := writeJSON("data/github.json", repos); err != nil {
		log.Fatalf("failed to create GitHub data file: %v", err)
	}

	tc := httptalks.NewClient("https://raw.githubusercontent.com/mdlayher/talks/master/talks.json")
	talks, err := tc.ListTalks(ctx)
	if err != nil {
		log.Fatalf("failed to get talks metadata: %v", err)
	}

	if err := writeJSON("data/talks.json", talks); err != nil {
		log.Fatalf("failed to create talks data file: %v", err)
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
