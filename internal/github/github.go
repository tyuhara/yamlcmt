package github

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/google/go-github/v66/github"
	"github.com/tyuhara/yamlcmt/internal/diff"
	"golang.org/x/oauth2"
)

// TemplateData represents data available in templates
type TemplateData struct {
	Summary      string
	Details      string
	HasChanges   bool
	Added        int
	Deleted      int
	Modified     int
	AddedList    []string
	DeletedList  []string
	ModifiedList []string
	Link         string
	Vars         map[string]interface{}
}

// getClient creates a GitHub client with the token from environment variable
func getClient(ctx context.Context) (*github.Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}

// parseRepo splits "owner/repo" into owner and repo
func parseRepo(repo string) (string, string, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %s (expected: owner/repo)", repo)
	}
	return parts[0], parts[1], nil
}

// PostComment posts a comment to a GitHub PR using go-github
func PostComment(repo string, prNumber int, body string) error {
	ctx := context.Background()
	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	owner, repoName, err := parseRepo(repo)
	if err != nil {
		return err
	}

	comment := &github.IssueComment{
		Body: github.String(body),
	}

	_, _, err = client.Issues.CreateComment(ctx, owner, repoName, prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Posted GitHub comment\n")
	return nil
}

// AddLabel adds a label to a GitHub PR using go-github
func AddLabel(repo string, prNumber int, label string) error {
	if label == "" {
		return nil
	}

	ctx := context.Background()
	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	owner, repoName, err := parseRepo(repo)
	if err != nil {
		return err
	}

	_, _, err = client.Issues.AddLabelsToIssue(ctx, owner, repoName, prNumber, []string{label})
	if err != nil {
		return fmt.Errorf("failed to add label: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Applied GitHub label: %s\n", label)
	return nil
}

// AddLabels adds multiple labels to a GitHub PR using go-github
func AddLabels(repo string, prNumber int, labels []string) error {
	if len(labels) == 0 {
		return nil
	}

	// Filter out empty labels
	validLabels := make([]string, 0, len(labels))
	for _, label := range labels {
		if label != "" {
			validLabels = append(validLabels, label)
		}
	}

	if len(validLabels) == 0 {
		return nil
	}

	ctx := context.Background()
	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	owner, repoName, err := parseRepo(repo)
	if err != nil {
		return err
	}

	_, _, err = client.Issues.AddLabelsToIssue(ctx, owner, repoName, prNumber, validLabels)
	if err != nil {
		return fmt.Errorf("failed to add labels: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Applied GitHub labels: %s\n", strings.Join(validLabels, ", "))
	return nil
}

// RenderTemplate renders a template with the given data
func RenderTemplate(tmplStr string, data TemplateData) (string, error) {
	tmpl, err := template.New("comment").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// PrepareTemplateData prepares template data from diff result
func PrepareTemplateData(result *diff.Result, details string, link string, vars map[string]interface{}) TemplateData {
	added := len(result.Added)
	deleted := len(result.Deleted)
	modified := len(result.Modified)

	summary := fmt.Sprintf("Plan: %d to add, %d to delete, %d to modify", added, deleted, modified)

	// Extract and sort keys
	addedList := make([]string, 0, len(result.Added))
	for k := range result.Added {
		addedList = append(addedList, k)
	}
	sort.Strings(addedList)

	deletedList := make([]string, 0, len(result.Deleted))
	for k := range result.Deleted {
		deletedList = append(deletedList, k)
	}
	sort.Strings(deletedList)

	modifiedList := make([]string, 0, len(result.Modified))
	for k := range result.Modified {
		modifiedList = append(modifiedList, k)
	}
	sort.Strings(modifiedList)

	return TemplateData{
		Summary:      summary,
		Details:      details,
		HasChanges:   result.HasDifferences(),
		Added:        added,
		Deleted:      deleted,
		Modified:     modified,
		AddedList:    addedList,
		DeletedList:  deletedList,
		ModifiedList: modifiedList,
		Link:         link,
		Vars:         vars,
	}
}
