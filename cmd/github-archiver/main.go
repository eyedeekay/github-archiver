package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/eyedeekay/github-archiver/pkg/analyzer"
	"github.com/eyedeekay/github-archiver/pkg/archiver"
	"github.com/eyedeekay/github-archiver/pkg/github"
	"github.com/eyedeekay/github-archiver/pkg/logger"
	"github.com/eyedeekay/github-archiver/pkg/util"
)

func main() {
	// Define command-line flags
	token := flag.String("token", "", "GitHub personal access token")
	target := flag.String("target", "", "GitHub username or organization name")
	dryRun := flag.Bool("dry-run", false, "Perform a dry run without making changes")
	org := flag.Bool("org", false, "Work on a github organization")
	inactivityThreshold := flag.Int("threshold", 2, "Inactivity threshold in years")
	verbose := flag.Bool("verbose", false, "Enable verbose (debug) logging")
	quiet := flag.Bool("quiet", false, "Show only warnings and errors")
	force := flag.Bool("force", false, "Force processing even if errors occur")
	flag.Parse()
	util.FORCE_PROCESSING = *force

	// Configure logging level
	if *verbose {
		logger.SetDefaultLevel(logger.DebugLevel)
		logger.Debug("Debug logging enabled")
	} else if *quiet {
		logger.SetDefaultLevel(logger.WarnLevel)
	}

	// Validate required flags
	if *token == "" || *target == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Create a context that can be canceled
	ctx := context.Background()

	// Initialize GitHub client
	logger.Debug("Initializing GitHub client")
	client, err := github.NewClient(ctx, *token)
	if util.ForceProcessing(err) {
		logger.Fatal("Failed to create GitHub client: %v", err)
	}

	// Create the repository analyzer
	repoAnalyzer := analyzer.NewAnalyzer(client, time.Duration(*inactivityThreshold)*365*24*time.Hour)
	logger.Debug("Repository analyzer initialized with %d year threshold", *inactivityThreshold)

	// Create the repository archiver
	repoArchiver := archiver.NewArchiver(client)
	logger.Debug("Repository archiver initialized")

	// 1. Fetch all repositories for the target
	logger.Info("Fetching repositories for %s...", *target)
	repos, err := client.ListRepositories(ctx, *target, *org)
	if util.ForceProcessing(err) {
		logger.Fatal("Failed to list repositories: %v", err)
	}
	logger.Info("Found %d repositories for %s", len(repos), *target)

	// 2. Analyze repositories for inactivity
	logger.Info("Analyzing repository activity...")
	inactiveRepos, err := repoAnalyzer.FindInactiveRepositories(ctx, repos)
	if util.ForceProcessing(err) {
		logger.Fatal("Failed to analyze repositories: %v", err)
	}

	if len(inactiveRepos) == 0 {
		logger.Info("No inactive repositories found.")
		return
	}

	logger.Info("%d repositories inactive for %d+ years:", len(inactiveRepos), *inactivityThreshold)
	for _, repo := range inactiveRepos {
		logger.Info("  - %s (Last activity: %s)", repo.Name, repo.LastActivity.Format("2006-01-02"))
	}

	// Stop here if this is a dry run
	if *dryRun {
		logger.Info("Dry run completed. No changes were made.")
		return
	}

	// 3. Archive inactive repositories
	logger.Info("Archiving %d repositories:", len(inactiveRepos))
	archiveNamespace := fmt.Sprintf("%s-archive", *target)

	for i, repo := range inactiveRepos {
		logger.Info("  - [%d/%d] Processing repository %s", i+1, len(inactiveRepos), repo.Name)

		err := repoArchiver.ArchiveRepository(ctx, *target, archiveNamespace, repo.Name)
		if util.ForceProcessing(err) {
			logger.Error("Failed to archive repository %s: %v", repo.Name, err)
			continue
		}
		logger.Info("  - [%d/%d] Successfully archived %s", i+1, len(inactiveRepos), repo.Name)
	}

	logger.Info("Archive process completed. %d repositories archived.", len(inactiveRepos))
}
