package main

import (
	"fmt"
	"log"

	"syac/internal/version"
	"syac/pkg/gitlab"
)

func main() {
	client, err := gitlab.NewClient()
	if err != nil {
		log.Fatalf("[gitlab] init failed: %v", err)
	}

	// (Optional) Show the latest semantic tag in the repo
	latest, err := client.Tags.GetLatestTag()
	if err != nil {
		log.Fatalf("[gitlab] get latest tag failed: %v", err)
	}
	fmt.Printf("latest tag: %s\n", latest.String())

	// Choose a bump. Use the enum, or parse a string if you prefer.
	bump := version.Patch
	// If you must start from a string:
	// bump, err = version.ParseVersionType("patch")
	// if err != nil { log.Fatalf("invalid bump: %v", err) }

	current, next, err := client.Tags.GetNextVersion(bump)
	if err != nil {
		log.Fatalf("[gitlab] failed to get next version: %v", err)
	}

	fmt.Printf("current=%s next=%s (bump=%s)\n", current.String(), next.String(), bump.String())
}
