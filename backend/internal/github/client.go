package github

import (
	"archive/tar"
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

	// Clone the repository
	cmd := exec.Command("git", "clone", repoURL, cloneDir)
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
	// Get the directory containing the Dockerfile. This will be the root of our tar archive.
	dockerfileDir := filepath.Dir(dockerfilePath)

	// Create a new tar writer
	tw := tar.NewWriter(writer)
	defer tw.Close()

	// Walk through the directory containing the Dockerfile
	err := filepath.Walk(dockerfileDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		// Get the relative path from the dockerfileDir to the current file/dir
		relPath, err := filepath.Rel(dockerfileDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// If relPath is ".", it's the dockerfileDir itself, which we don't need to add as a separate entry.
		if relPath == "." {
			return nil
		}

		// Update the header name to use forward slashes and relative path
		header.Name = strings.ReplaceAll(relPath, "\\", "/")

		// Write the header
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// If it's a regular file, write its contents
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return fmt.Errorf("failed to write file contents to tar: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create tar archive from %s: %w", dockerfileDir, err)
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
