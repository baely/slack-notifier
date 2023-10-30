package github

import (
	"context"
	"github.com/baely/slack-notifier/internal/slack"
	"github.com/baely/slack-notifier/pkg/set"
	"github.com/google/go-github/v56/github"
	"log"
	"strings"
	"sync"
	"time"
)

type Client struct {
	*github.Client

	owner, repo, slackWebhook string
}

func NewGHClient(token, repo, slackWebhook string) *Client {
	client := github.NewClient(nil).WithAuthToken(token)

	owner := repo[:strings.Index(repo, "/")]
	repo = repo[strings.Index(repo, "/")+1:]

	return &Client{
		Client:       client,
		owner:        owner,
		repo:         repo,
		slackWebhook: slackWebhook,
	}
}

func (c *Client) handleActionCompletion(check *github.CheckRun) {
	if !(check.GetConclusion() == "failure" || check.GetConclusion() == "timed_out") {
		return
	}

	log.Printf("Check %s failed", check.GetName())

	slack.PostMessage(c.slackWebhook, "failed")
}

func (c *Client) waitForAction(wg *sync.WaitGroup, check *github.CheckRun) {
	var cr *github.CheckRun
	var err error
	for {
		cr, _, err = c.Checks.GetCheckRun(context.Background(), c.owner, c.repo, check.GetID())
		if err != nil {
			log.Printf("Error while getting check run: %s", err)
			return
		}

		if cr.GetStatus() == "completed" {
			break
		}

		time.Sleep(5 * time.Second)
	}
	c.handleActionCompletion(cr)
	wg.Done()
}

func (c *Client) WaitForActions(sha, requiredChecksRaw string) {
	time.Sleep(5 * time.Second)

	requiredChecks := parseRequiredChecks(requiredChecksRaw)

	ctx := context.Background()
	crs, _, err := c.Client.Checks.ListCheckRunsForRef(ctx, c.owner, c.repo, sha, nil)

	if err != nil {
		log.Printf("Error while getting check runs: %s", err)
		return
	}

	if crs.GetTotal() == 0 {
		log.Println("No check runs found")
		return
	}

	wg := sync.WaitGroup{}

	for _, cr := range crs.CheckRuns {
		if !requiredChecks.Contains(cr.GetName()) {
			continue
		}

		wg.Add(1)
		go c.waitForAction(&wg, cr)
	}

	wg.Wait()
}

func parseRequiredChecks(requiredChecks string) set.Set[string] {
	checks := strings.Split(requiredChecks, ",")
	checksSet := set.NewSet[string]()

	for _, check := range checks {
		checksSet.Add(strings.TrimSpace(check))
	}

	return checksSet
}
