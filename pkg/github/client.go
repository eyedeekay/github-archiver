package github

import (
	"context"
	"fmt"
	"time"

	"github.com/eyedeekay/github-archiver/pkg/logger"
	"github.com/eyedeekay/github-archiver/pkg/util"
	"github.com/google/go-github/v59/github"
	"golang.org/x/oauth2"
)

// Repository represents a GitHub repository with activity information
type Repository struct {
	Owner        string
	Name         string
	LastActivity time.Time
	IsArchived   bool
}

// Client wraps the GitHub API client
type Client struct {
	client *github.Client
}

// NewClient creates a new GitHub client with the provided token
func NewClient(ctx context.Context, token string) (*Client, error) {
	logger.Debug("Creating new GitHub client")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return &Client{
		client: github.NewClient(tc),
	}, nil
}

// ListRepositories fetches all repositories for a user or organization
func (c *Client) ListRepositories(ctx context.Context, target string, org bool) ([]Repository, error) {
	var allRepos []*github.Repository
	entityType := "user"
	if org {
		entityType = "organization"
	}

	logger.Info("Fetching repositories for %s %s", entityType, target)

	if !org {
		opts := &github.RepositoryListOptions{
			ListOptions: github.ListOptions{PerPage: 100},
		}

		for {
			logger.Debug("Fetching page %d of user repositories", opts.Page+1)
			repos, resp, err := c.client.Repositories.List(ctx, target, opts)
			if util.ForceProcessing(err) {
				logger.Error("Failed to list repositories for user %s: %v", target, err)
				return nil, fmt.Errorf("failed to list repositories: %w", err)
			}

			logger.Debug("Retrieved %d repositories on page %d", len(repos), opts.Page+1)
			allRepos = append(allRepos, repos...)

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	} else {
		opts := &github.RepositoryListByOrgOptions{
			ListOptions: github.ListOptions{PerPage: 100},
		}

		for {
			logger.Debug("Fetching page %d of organization repositories", opts.Page+1)
			repos, resp, err := c.client.Repositories.ListByOrg(ctx, target, opts)
			if util.ForceProcessing(err) {
				logger.Error("Failed to list repositories for organization %s: %v", target, err)
				return nil, fmt.Errorf("failed to list repositories: %w", err)
			}

			logger.Debug("Retrieved %d repositories on page %d", len(repos), opts.Page+1)
			allRepos = append(allRepos, repos...)

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}

	logger.Debug("Processing %d repositories", len(allRepos))
	result := make([]Repository, 0, len(allRepos))
	for _, repo := range allRepos {
		if repo == nil || repo.Name == nil || repo.Owner == nil || repo.Owner.Login == nil {
			logger.Warn("Skipping repository with incomplete data")
			continue
		}
		result = append(result, Repository{
			Owner:      *repo.Owner.Login,
			Name:       *repo.Name,
			IsArchived: repo.GetArchived(),
			// We'll get the actual last activity in the analyzer
			LastActivity: repo.GetUpdatedAt().Time,
		})
	}

	logger.Info("Successfully retrieved %d valid repositories for %s", len(result), target)
	return result, nil
}

// GetLastActivity fetches the latest activity timestamp for a repository
func (c *Client) GetLastActivity(ctx context.Context, owner, repo string) (time.Time, error) {
	logger.Debug("Fetching last activity for %s/%s", owner, repo)

	// Get repository information
	repository, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if util.ForceProcessing(err) {
		logger.Error("Failed to get repository info for %s/%s: %v", owner, repo, err)
		return time.Time{}, fmt.Errorf("failed to get repository info: %w", err)
	}

	// Start with the last push date
	lastActivity := repository.GetPushedAt().Time
	logger.Debug("Last push for %s/%s: %s", owner, repo, lastActivity.Format("2006-01-02"))

	// Check for more recent issues/PRs
	issueOpts := &github.IssueListByRepoOptions{
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	}

	logger.Debug("Checking for more recent issue/PR activity in %s/%s", owner, repo)
	issues, _, err := c.client.Issues.ListByRepo(ctx, owner, repo, issueOpts)
	if err == nil && len(issues) > 0 {
		issueTime := issues[0].GetUpdatedAt()
		logger.Debug("Most recent issue/PR activity: %s", issueTime.Format("2006-01-02"))
		if issueTime.After(lastActivity) {
			logger.Debug("Found more recent activity in issues/PRs: %s", issueTime.Format("2006-01-02"))
			lastActivity = issueTime.Time
		}
	} else if util.ForceProcessing(err) {
		logger.Warn("Error checking issues/PRs for %s/%s: %v", owner, repo, err)
	}

	logger.Debug("Final last activity date for %s/%s: %s", owner, repo, lastActivity.Format("2006-01-02"))
	return lastActivity, nil
}

// CreateArchiveNamespace checks if the archive organization/user exists
func (c *Client) CreateArchiveNamespace(ctx context.Context, namespace string) error {
	logger.Debug("Checking if archive namespace %s exists", namespace)

	// Check if the namespace exists as an organization
	logger.Debug("Checking if %s exists as an organization", namespace)
	_, _, err := c.client.Organizations.Get(ctx, namespace)
	if err == nil {
		logger.Debug("Archive namespace %s exists as an organization", namespace)
		return nil
	}

	// Check if the namespace exists as a user
	logger.Debug("Checking if %s exists as a user", namespace)
	_, _, err = c.client.Users.Get(ctx, namespace)
	if err == nil {
		logger.Debug("Archive namespace %s exists as a user", namespace)
		return nil
	}

	logger.Error("Archive namespace %s does not exist", namespace)
	// The GitHub API doesn't support programmatic creation of organizations
	return fmt.Errorf("archive namespace '%s' does not exist and cannot be created automatically. Please create the organization or user account manually", namespace)
}

// ForkRepository forks a repository to the archive namespace
func (c *Client) ForkRepository(ctx context.Context, owner, repo, targetOrg string) error {
	logger.Debug("Checking if %s/%s already exists", targetOrg, repo)

	// Check if repository already exists in target org
	_, _, err := c.client.Repositories.Get(ctx, targetOrg, repo)
	if err == nil {
		// Repository already exists in target org
		logger.Info("Repository %s/%s already exists, skipping fork creation", targetOrg, repo)
		return nil
	}

	logger.Debug("Forking %s/%s to %s", owner, repo, targetOrg)

	forkOpts := &github.RepositoryCreateForkOptions{
		Organization: targetOrg,
	}

	_, _, err = c.client.Repositories.CreateFork(ctx, owner, repo, forkOpts)
	if util.ForceProcessing(err) {
		logger.Error("Failed to fork %s/%s to %s: %v", owner, repo, targetOrg, err)
		if err != nil {
			return fmt.Errorf("failed to fork repository: %w", err)
		}
	}

	logger.Debug("Successfully forked %s/%s to %s", owner, repo, targetOrg)
	return nil
}

// DeleteRepository deletes a repository
func (c *Client) DeleteRepository(ctx context.Context, owner, repo string) error {
	logger.Debug("Deleting repository %s/%s", owner, repo)

	_, err := c.client.Repositories.Delete(ctx, owner, repo)
	if util.ForceProcessing(err) {
		logger.Error("Failed to delete repository %s/%s: %v", owner, repo, err)
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	logger.Debug("Successfully deleted repository %s/%s", owner, repo)
	return nil
}

// SetArchiveStatus marks a repository as archived
func (c *Client) SetArchiveStatus(ctx context.Context, owner, repo string, archived bool) error {
	action := "archive"
	if !archived {
		action = "unarchive"
	}

	logger.Debug("%s repository %s/%s", action, owner, repo)

	repository, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if util.ForceProcessing(err) {
		logger.Error("Failed to get repository info for %s/%s: %v", owner, repo, err)
		return fmt.Errorf("failed to get repository info: %w", err)
	}

	if repository == nil {
		logger.Error("Repository object for %s/%s is nil", owner, repo)
		return fmt.Errorf("repository object is nil")
	}

	repository.Archived = github.Bool(archived)

	_, _, err = c.client.Repositories.Edit(ctx, owner, repo, repository)
	if util.ForceProcessing(err) {
		logger.Error("Failed to %s repository %s/%s: %v", action, owner, repo, err)
		return fmt.Errorf("failed to update archive status: %w", err)
	}

	logger.Debug("Successfully %sd repository %s/%s", action, owner, repo)
	return nil
}
