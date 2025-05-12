package archiver

import (
	"context"
	"fmt"
	"time"

	"github.com/eyedeekay/github-archiver/pkg/github"
	"github.com/eyedeekay/github-archiver/pkg/logger"
	"github.com/eyedeekay/github-archiver/pkg/util"
)

// Archiver handles the repository archiving process
type Archiver struct {
	client *github.Client
}

// NewArchiver creates a new repository archiver
func NewArchiver(client *github.Client) *Archiver {
	return &Archiver{
		client: client,
	}
}

// ArchiveRepository archives a repository by:
// 1. Creating an archive namespace if it doesn't exist
// 2. Forking the repository to the archive namespace
// 3. Deleting the original repository
// 4. Setting the archived status to true on the forked repository
func (a *Archiver) ArchiveRepository(ctx context.Context, owner, archiveNamespace, repo string) error {
	logger.Debug("Beginning archive process for repository %s/%s", owner, repo)

	// 1. Create archive namespace if it doesn't exist
	logger.Info("Creating archive namespace %s...", archiveNamespace)
	err := a.client.CreateArchiveNamespace(ctx, archiveNamespace)
	if util.ForceProcessing(err) {
		logger.Error("Failed to create archive namespace %s: %v", archiveNamespace, err)
		return fmt.Errorf("failed to create archive namespace: %w", err)
	}
	logger.Debug("Archive namespace %s confirmed", archiveNamespace)

	// 2. Fork the repository to the archive namespace
	logger.Info("Forking %s/%s to %s...", owner, repo, archiveNamespace)
	err = a.client.ForkRepository(ctx, owner, repo, archiveNamespace)
	if util.ForceProcessing(err) {
		logger.Error("Failed to fork repository %s/%s: %v", owner, repo, err)
		return fmt.Errorf("failed to fork repository: %w", err)
	}
	logger.Debug("Repository forked successfully")

	// Wait for the fork to be created
	waitTime := 5 * time.Second
	logger.Debug("Waiting %v for fork to complete...", waitTime)
	time.Sleep(waitTime)

	// 3. Delete the original repository
	logger.Info("Deleting original repository %s/%s...", owner, repo)
	err = a.client.DeleteRepository(ctx, owner, repo)
	if util.ForceProcessing(err) {
		logger.Error("Failed to delete original repository %s/%s: %v", owner, repo, err)
		return fmt.Errorf("failed to delete original repository: %w", err)
	}
	logger.Debug("Original repository deleted")

	// 4. Set the archived status to true on the forked repository
	logger.Info("Setting archived status on %s/%s...", archiveNamespace, repo)
	err = a.client.SetArchiveStatus(ctx, archiveNamespace, repo, true)
	if util.ForceProcessing(err) {
		logger.Error("Failed to set archived status on %s/%s: %v", archiveNamespace, repo, err)
		return fmt.Errorf("failed to set archived status: %w", err)
	}
	logger.Debug("Archive status set successfully")

	logger.Info("Repository %s successfully archived to %s/%s", repo, archiveNamespace, repo)
	return nil
}
