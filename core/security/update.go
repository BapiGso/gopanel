package security

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type release struct {
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// getLatestReleaseDate fetches the latest release date and download URL for the specified repository.
func getLatestReleaseDate() (time.Time, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", "BapiGso", "gopanel")
	resp, err := http.Get(url)
	if err != nil {
		return time.Time{}, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return time.Time{}, "", err
	}

	var rel release
	if err := json.Unmarshal(body, &rel); err != nil {
		return time.Time{}, "", err
	}

	// Construct download URL based on OS and architecture
	osLower := runtime.GOOS
	goArch := runtime.GOARCH
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/gopanel_%s_%s",
		"BapiGso", "gopanel", osLower, goArch)

	return rel.PublishedAt, downloadURL, nil
}

// updateBinaryIfNeeded updates the local binary file if a newer version is available.
func updateBinaryIfNeeded() error {
	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot get executable path: %w", err)
	}

	// Get absolute path to the binary
	localBinaryPath, err := filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("cannot get absolute path: %w", err)
	}

	// Step 1: Get the last modified time of the local binary
	localFileInfo, err := os.Stat(localBinaryPath)
	if err != nil {
		return fmt.Errorf("error stating local binary: %w", err)
	}
	localFileModTime := localFileInfo.ModTime()

	// Step 2: Get the latest release date from GitHub
	latestReleaseDate, downloadURL, err := getLatestReleaseDate()
	if err != nil {
		return fmt.Errorf("error fetching latest release date: %w", err)
	}

	// Step 3: Compare release date with local binary mod time, and update if needed
	if latestReleaseDate.After(localFileModTime) {
		fmt.Println("Newer release found. Downloading updated binary...")

		// Download the new binary
		resp, err := http.Get(downloadURL)
		if err != nil {
			return fmt.Errorf("error downloading binary: %w", err)
		}
		defer resp.Body.Close()

		// Create or overwrite the local binary file
		out, err := os.Create(localBinaryPath)
		if err != nil {
			return fmt.Errorf("error creating local binary file: %w", err)
		}
		defer out.Close()

		// Write the downloaded content to the file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return fmt.Errorf("error writing to local binary file: %w", err)
		}

		fmt.Println("Binary updated successfully.")
	} else {
		return fmt.Errorf("local binary is up-to-date")
	}

	return nil
}
