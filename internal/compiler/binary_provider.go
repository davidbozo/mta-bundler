package compiler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// BinaryProvider defines the strategy interface for obtaining luac_mta binary
type BinaryProvider interface {
	GetBinary() (string, error)
	Name() string
}

// LocalBinaryProvider searches for binary in local filesystem
type LocalBinaryProvider struct{}

// NewLocalBinaryProvider creates a new local binary provider
func NewLocalBinaryProvider() LocalBinaryProvider {
	return LocalBinaryProvider{}
}

// Name returns the provider name
func (p LocalBinaryProvider) Name() string {
	return "local"
}

// GetBinary attempts to find the luac_mta binary locally
func (p LocalBinaryProvider) GetBinary() (string, error) {
	var candidates []string

	// Platform-specific binary names
	if runtime.GOOS == "windows" {
		candidates = []string{
			"luac_mta.exe",
			"./luac_mta.exe",
			"./bin/luac_mta.exe",
			"C:\\bin\\luac_mta.exe",
		}
	} else {
		candidates = []string{
			"luac_mta",
			"./luac_mta",
			"./bin/luac_mta",
			"/usr/local/bin/luac_mta",
			"/usr/bin/luac_mta",
		}
	}

	// Check PATH first
	if path, err := exec.LookPath("luac_mta"); err == nil {
		return path, nil
	}

	// Check candidate locations
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("luac_mta binary not found in PATH or common locations")
}

// WebBinaryProvider downloads binary from MTA servers
type WebBinaryProvider struct{}

// NewWebBinaryProvider creates a new web binary provider
func NewWebBinaryProvider() WebBinaryProvider {
	return WebBinaryProvider{}
}

// Name returns the provider name
func (p WebBinaryProvider) Name() string {
	return "web"
}

// GetBinary downloads and returns the luac_mta binary from MTA servers
func (p WebBinaryProvider) GetBinary() (string, error) {
	url, filename, err := p.getBinaryURL()
	if err != nil {
		return "", fmt.Errorf("failed to determine binary URL: %w", err)
	}

	// Use system temp directory
	tempDir := os.TempDir()
	binaryPath := filepath.Join(tempDir, filename)

	// Check if already downloaded
	if _, err := os.Stat(binaryPath); err == nil {
		fmt.Printf("Found existing %s binary in temp directory: %s\n", runtime.GOOS, binaryPath)
		return binaryPath, nil
	}

	fmt.Printf("Downloading %s binary from MTA servers to temp directory...\n", runtime.GOOS)

	// Download the binary
	if err := p.downloadFile(url, binaryPath); err != nil {
		return "", fmt.Errorf("failed to download binary: %w", err)
	}

	// Make binary executable on Unix-like systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return "", fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	fmt.Printf("Binary downloaded successfully: %s\n", binaryPath)
	return binaryPath, nil
}

// getBinaryURL returns the download URL and filename based on the current OS and architecture
func (p WebBinaryProvider) getBinaryURL() (string, string, error) {
	switch runtime.GOOS {
	case "windows":
		return "https://luac.mtasa.com/files/windows/x86/luac_mta.exe", "luac_mta.exe", nil
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "https://luac.mtasa.com/files/linux/x64/luac_mta", "luac_mta", nil
		case "386":
			return "https://luac.mtasa.com/files/linux/x86/luac_mta", "luac_mta", nil
		default:
			return "", "", fmt.Errorf("unsupported Linux architecture: %s", runtime.GOARCH)
		}
	default:
		return "", "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// downloadFile downloads a file from the given URL to the specified path
func (p WebBinaryProvider) downloadFile(url, filepath string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Copy data to file
	_, err = io.Copy(out, resp.Body)
	return err
}
