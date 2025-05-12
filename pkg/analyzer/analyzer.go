package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/eyedeekay/github-archiver/pkg/github"
	"github.com/eyedeekay/github-archiver/pkg/logger"
	"github.com/eyedeekay/github-archiver/pkg/util"
)

// Analyzer identifies inactive repositories
type Analyzer struct {
	client           *github.Client
	inactivityPeriod time.Duration
}

// NewAnalyzer creates a new repository analyzer
func NewAnalyzer(client *github.Client, inactivityPeriod time.Duration) *Analyzer {
	return &Analyzer{
		client:           client,
		inactivityPeriod: inactivityPeriod,
	}
}

// FindInactiveRepositories identifies repositories with no activity
// within the defined inactivity period
func (a *Analyzer) FindInactiveRepositories(ctx context.Context, repos []github.Repository) ([]github.Repository, error) {
	var inactiveRepos []github.Repository

	now := time.Now()
	cutoffDate := now.Add(-a.inactivityPeriod)
	logger.Debug("Inactivity threshold set to %v (before %s)", a.inactivityPeriod, cutoffDate.Format("2006-01-02"))

	logger.Info("Analyzing %d repositories for inactivity", len(repos))

	for i, repo := range repos {
		logger.Debug("[%d/%d] Checking repository %s/%s", i+1, len(repos), repo.Owner, repo.Name)

		// Skip already archived repositories
		if repo.IsArchived {
			logger.Debug("Skipping %s/%s - already archived", repo.Owner, repo.Name)
			continue
		}

		// Get the latest activity timestamp
		logger.Debug("Fetching last activity for %s/%s", repo.Owner, repo.Name)
		lastActivity, err := a.client.GetLastActivity(ctx, repo.Owner, repo.Name)
		if util.ForceProcessing(err) {
			logger.Error("Failed to check activity for %s/%s: %v", repo.Owner, repo.Name, err)
			return nil, fmt.Errorf("failed to check activity for %s/%s: %w", repo.Owner, repo.Name, err)
		}

		// Add repository details to the result
		repo.LastActivity = lastActivity

		// Format the duration since last activity for logging
		inactiveDuration := now.Sub(lastActivity).Round(24 * time.Hour)

		// Check if the repository is inactive
		if lastActivity.Before(cutoffDate) {
			logger.Debug("Repository %s/%s is inactive (last activity: %s, %v ago)",
				repo.Owner, repo.Name, lastActivity.Format("2006-01-02"), inactiveDuration)
			inactiveRepos = append(inactiveRepos, repo)
		} else {
			logger.Debug("Repository %s/%s is active (last activity: %s, %v ago)",
				repo.Owner, repo.Name, lastActivity.Format("2006-01-02"), inactiveDuration)
		}

		// Add a small delay to prevent rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("Found %d inactive repositories out of %d total", len(inactiveRepos), len(repos))
	return inactiveRepos, nil
}
