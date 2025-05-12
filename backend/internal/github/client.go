package github

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitClient represents a client for interacting with Git repositories
type GitClient struct {
	TempDir string // Directory to clone repositories into
}

// NewGitClient creates a new Git client
func NewGitClient(tempDir string) (*GitClient, error) {
	// Create temp directory if it doesn't exist
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &GitClient{
		TempDir: tempDir,
	}, nil
}

// CloneRepository clones a GitHub repository to a local directory and returns the path
func (gc *GitClient) CloneRepository(repoURL string) (string, error) {
	// Validate GitHub URL
	if !isValidGitHubURL(repoURL) {
		return "", fmt.Errorf("invalid GitHub repository URL: %s", repoURL)
	}

	// Create a unique directory name based on the repository URL
	repoName := getRepoNameFromURL(repoURL)
	if repoName == "" {
		return "", fmt.Errorf("could not extract repository name from URL: %s", repoURL)
	}

	// Create a unique temp directory for this clone
	cloneDir := filepath.Join(gc.TempDir, fmt.Sprintf("%s-%d", repoName, os.Getpid()))
	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create clone directory: %w", err)
	}

	// Clone the repository with --depth 1 for faster cloning
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, cloneDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed: %s: %w", string(output), err)
	}

	return cloneDir, nil
}

// FindDockerfile searches for a Dockerfile in the repository
func (gc *GitClient) FindDockerfile(repoPath string) (string, error) {
	var dockerfilePath string
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "Dockerfile" {
			dockerfilePath = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error searching for Dockerfile: %w", err)
	}
	if dockerfilePath == "" {
		return "", fmt.Errorf("no Dockerfile found in the repository")
	}
	return dockerfilePath, nil
}

// PrepareDockerBuildContext creates a tar archive from the directory containing the Dockerfile.
// dockerfilePath should be the absolute path to the Dockerfile.
func (gc *GitClient) PrepareDockerBuildContext(dockerfilePath string, writer io.Writer) error {
	// Get the directory containing the Dockerfile
	dockerfileDir := filepath.Dir(dockerfilePath)

	// Create git archive command
	cmd := exec.Command("git", "archive", "--format=tar", "HEAD")
	cmd.Dir = dockerfileDir // Set the working directory to the repository root

	// Create a pipe to capture the output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start git archive: %w", err)
	}

	// Copy the git archive output to our writer
	if _, err := io.Copy(writer, stdout); err != nil {
		cmd.Wait() // Clean up the command
		return fmt.Errorf("failed to copy git archive output: %w", err)
	}

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git archive failed: %w", err)
	}

	return nil
}

// CleanupRepository removes the cloned repository directory
func (gc *GitClient) CleanupRepository(repoPath string) error {
	return os.RemoveAll(repoPath)
}

// isValidGitHubURL checks if a URL is a valid GitHub repository URL
func isValidGitHubURL(url string) bool {
	// Basic validation for GitHub URLs
	return strings.HasPrefix(url, "https://github.com/") ||
		strings.HasPrefix(url, "git@github.com:") ||
		strings.HasPrefix(url, "http://github.com/")
}

// getRepoNameFromURL extracts the repository name from a GitHub URL
func getRepoNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) < 1 {
		return ""
	}

	// Get the last part of the URL and remove .git extension if present
	repoName := parts[len(parts)-1]
	return strings.TrimSuffix(repoName, ".git")
}
