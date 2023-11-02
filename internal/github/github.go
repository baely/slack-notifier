package github

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/go-github/v56/github"
	"golang.org/x/sync/errgroup"

	"github.com/baely/slack-notifier/internal/slack"
	"github.com/baely/slack-notifier/pkg/set"
)

type Client struct {
	*github.Client

	owner, repo, slackWebhook string
}

// NewClient creates a new GitHub client
func NewClient(token, repo, slackWebhook string) *Client {
	client := github.NewClient(nil).WithAuthToken(token)

	return &Client{
		Client:       client,
		owner:        strings.SplitN(repo, "/", 2)[0],
		repo:         strings.SplitN(repo, "/", 2)[1],
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
func (c *Client) waitForAction(check *github.CheckRun) error {
	for _ = range time.Tick(5 * time.Second) {
		cr, _, err := c.Checks.GetCheckRun(context.Background(), c.owner, c.repo, check.GetID())
		if err != nil {
			log.Printf("Error while getting check run: %s", err)
			return err
		}

		log.Printf("Check %s status: %s", cr.GetName(), cr.GetStatus())

		if cr.GetStatus() == "completed" {
			c.handleActionCompletion(cr)
			return nil
		}
	}
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
	errGroup := &errgroup.Group{}
	for _, cr := range checkRuns {
		if requiredChecks.Len() > 0 && !requiredChecks.Contains(cr.GetName()) {
			continue
		}
		errGroup.Go(func() error {
			return c.waitForAction(cr)
		})
	}
	return errGroup.Wait()
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
