package main

import (
	"fmt"
	"os"

	"github.com/AdeptMind/pr-emojis-in-slack/internal/config"
	"github.com/AdeptMind/pr-emojis-in-slack/internal/github"
	"github.com/AdeptMind/pr-emojis-in-slack/internal/run"
	"github.com/AdeptMind/pr-emojis-in-slack/internal/slack"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	ghBackend := github.NewWebBackend(cfg.GithubEventPath, cfg.GithubRepo, cfg.GithubToken)
	ghClient := github.NewClient(ghBackend)

	slBackend := slack.NewWebBackend(cfg.SlackAPIToken)
	slClient := slack.NewClient(slBackend)

	if err := run.Run(&cfg, ghClient, slClient); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
