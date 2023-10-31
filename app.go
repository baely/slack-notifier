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
	if githubToken == "" || githubRepo == "" || commitSha == "" || slackWebhook == "" {
		log.Println("Missing required input variables")
		os.Exit(1)
	}

	log.Println("Starting action")

	ghClient := github.NewGHClient(githubToken, githubRepo, slackWebhook)

	err := ghClient.WaitForActions(commitSha, requiredChecks)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
