package main

import (
	"log"
	"os"

	"github.com/baely/slack-notifier/internal/github"
)

var (
	githubToken    = os.Getenv("INPUT_GITHUB_TOKEN")
	githubRepo     = os.Getenv("INPUT_GITHUB_REPO")
	commitSha      = os.Getenv("INPUT_COMMIT_SHA")
	slackWebhook   = os.Getenv("INPUT_SLACK_WEBHOOK")
	requiredChecks = os.Getenv("INPUT_REQUIRED_CHECKS")
)

func main() {
	if githubToken == "" || slackWebhook == "" {
		log.Println("Missing required input variables")
		os.Exit(1)
	}

	log.Println("Starting action")

	ghClient := github.NewGHClient(githubToken, githubRepo, slackWebhook)

	ghClient.WaitForActions(commitSha, requiredChecks)
}
