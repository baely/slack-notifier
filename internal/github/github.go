package github

import (
	"context"
	"fmt"
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

// NewClient creates a new GitHub client
func NewClient(token, repo, slackWebhook string) *Client {
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

// handleActionCompletion handles the completion of a GitHub action
func (c *Client) handleActionCompletion(check *github.CheckRun) {
	log.Printf("Check %s conclusion: %s", check.GetName(), check.GetConclusion())

	if !(check.GetConclusion() == "failure" || check.GetConclusion() == "timed_out") {
		return
	}

	commitUrl := fmt.Sprintf("https://github.com/%s/%s/commit/%s", c.owner, c.repo, check.GetHeadSHA())

	slack.PostMessage(c.slackWebhook, slack.Message{
		Name:       check.GetName(),
		Conclusion: check.GetConclusion(),
		CheckUrl:   check.GetHTMLURL(),
		CommitUrl:  commitUrl,
	})
}

// waitForAction waits for a GitHub action to complete and then calls handleActionCompletion to handle the completion
func (c *Client) waitForAction(wg *sync.WaitGroup, check *github.CheckRun) {
	var cr *github.CheckRun
	var err error
	for {
		cr, _, err = c.Checks.GetCheckRun(context.Background(), c.owner, c.repo, check.GetID())
		if err != nil {
			log.Printf("Error while getting check run: %s", err)
			wg.Done()
			return
		}

		log.Printf("Check %s status: %s", cr.GetName(), cr.GetStatus())

		if cr.GetStatus() == "completed" {
			break
		}

		time.Sleep(5 * time.Second)
	}
	c.handleActionCompletion(cr)
	wg.Done()
}

// getAllChecks gets all GitHub actions for a commit
func (c *Client) getAllChecks(sha string) ([]*github.CheckRun, error) {
	ctx := context.Background()
	crs := make([]*github.CheckRun, 0)
	crsFirst, resp, err := c.Client.Checks.ListCheckRunsForRef(ctx, c.owner, c.repo, sha, nil)
	if err != nil {
		log.Printf("Error while getting check runs: %s", err)
		return nil, err
	}

	if crsFirst.GetTotal() == 0 {
		log.Println("No check runs found")
		return crs, nil
	}

	crs = append(crs, crsFirst.CheckRuns...)

	for {
		if resp.NextPage == 0 {
			break
		}
		var crsNext *github.ListCheckRunsResults
		crsNext, resp, err = c.Client.Checks.ListCheckRunsForRef(ctx, c.owner, c.repo, sha, &github.ListCheckRunsOptions{
			ListOptions: github.ListOptions{
				Page: resp.NextPage,
			},
		})
		if err != nil {
			log.Printf("Error while getting check runs: %s", err)
			return nil, err
		}
		crs = append(crs, crsNext.CheckRuns...)
	}
	return crs, nil
}

// WaitForActions waits for all GitHub actions to complete
func (c *Client) WaitForActions(sha, requiredChecksRaw string) error {
	// Wait for all checks to at least be queued
	log.Println("Waiting 15 seconds for checks to be queued")
	time.Sleep(15 * time.Second)

	checkRuns, err := c.getAllChecks(sha)
	if err != nil {
		return err
	}

	requiredChecks := parseRequiredChecks(requiredChecksRaw)
	wg := sync.WaitGroup{}
	for _, cr := range checkRuns {
		if requiredChecks.Len() > 0 && !requiredChecks.Contains(cr.GetName()) {
			continue
		}
		wg.Add(1)
		go c.waitForAction(&wg, cr)
	}
	wg.Wait()

	return nil
}

// parseRequiredChecks parses the required checks from a comma-separated string
func parseRequiredChecks(requiredChecks string) set.Set[string] {
	checks := strings.Split(requiredChecks, ",")
	checksSet := set.NewSet[string]()

	for _, check := range checks {
		checksSet.Add(strings.TrimSpace(check))
	}

	return checksSet
}
