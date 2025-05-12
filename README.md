# GitHub Archiver

A tool to identify and archive inactive GitHub repositories based on a configurable inactivity threshold.

## Overview

GitHub Archiver automates the process of analyzing repositories for inactivity and archiving them. It:

1. Lists all repositories for a specified user or organization
2. Analyzes repository activity to identify inactive ones
3. Archives inactive repositories by:
   - Creating an archive namespace (if needed)
   - Forking to the archive namespace
   - Deleting the original repository
   - Marking the forked repository as archived

## Installation

```bash
go get github.com/eyedeekay/github-archiver
```

## Dependencies

- Go 1.x
- github.com/google/go-github/v59
- golang.org/x/oauth2

## Usage

```bash
github-archiver --token YOUR_GITHUB_TOKEN --target USERNAME [options]
```

### Options

- `--token`: GitHub personal access token (required)
- `--target`: GitHub username or organization (required)
- `--dry-run`: Analyze repositories without making changes
- `--org`: Specify if target is an organization (default: false)
- `--threshold`: Inactivity threshold in years (default: 2)
- `--verbose`: Enable verbose (debug) logging
- `--quiet`: Show only warnings and errors

## Example

To identify repositories inactive for 3+ years and perform archiving:

```bash
github-archiver --token ghp_xxxxxxxxxxxx --target myusername --threshold 3
```

For a dry run that only reports inactive repositories:

```bash
github-archiver --token ghp_xxxxxxxxxxxx --target myorg --org --dry-run
```

For detailed debug information:

```bash
github-archiver --token ghp_xxxxxxxxxxxx --target myusername --verbose
```

## Core Components

- **Client**: Wraps GitHub API functionality
- **Analyzer**: Identifies inactive repositories based on the inactivity threshold
- **Archiver**: Handles the repository archiving process
- **Logger**: Provides structured logging with multiple severity levels

The archive namespace requires manual creation for now.
Create it as `{target}-archive` (e.g., `username-archive`).