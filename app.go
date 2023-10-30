package main

import (
	"log"
	"os"
)

var (
	githubToken  = os.Getenv("INPUT_GITHUB_TOKEN")
	slackWebhook = os.Getenv("INPUT_SLACK_WEBHOOK")
)

func main() {
	if githubToken == "" || slackWebhook == "" {
		log.Println("Missing required input variables")
		os.Exit(1)
	}

}
