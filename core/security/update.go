package security

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

const repoSlug = "BapiGso/gopanel"

type release struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getLatestReleaseDate() (time.Time, string, error) {
	if rel, found, err := selfupdate.DetectLatest(repoSlug); err == nil && found && rel.PublishedAt != nil && rel.AssetURL != "" {
		return rel.PublishedAt.UTC(), rel.AssetURL, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repoSlug)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Get(url)
	if err != nil {
		return time.Time{}, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, "", fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return time.Time{}, "", err
	}

	var rel release
	if err := json.Unmarshal(body, &rel); err != nil {
		return time.Time{}, "", err
	}

	downloadURL, err := pickReleaseAssetURL(rel.Assets)
	if err != nil {
		return time.Time{}, "", err
	}

	return rel.PublishedAt, downloadURL, nil
}

func updateBinaryIfNeeded() error {
	currentVersion, ok := currentVersion()
	if !ok {
		return fmt.Errorf("current build version is unavailable")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot get executable path: %w", err)
	}
	localBinaryPath, err := filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("cannot get absolute path: %w", err)
	}

	if rel, found, err := selfupdate.DetectLatest(repoSlug); err == nil && found {
		if !rel.Version.GT(currentVersion) {
			return fmt.Errorf("latest release version is %s. Local binary is up-to-date", rel.Version)
		}
		if err := selfupdate.UpdateTo(rel.AssetURL, localBinaryPath); err != nil {
			return fmt.Errorf("error replacing binary file: %w", err)
		}
		return nil
	}

	latestVersion, downloadURL, err := getLatestReleaseVersion()
	if err != nil {
		return fmt.Errorf("error fetching latest release: %w", err)
	}
	if !latestVersion.GT(currentVersion) {
		return fmt.Errorf("latest release version is %s. Local binary is up-to-date", latestVersion)
	}
	if err := selfupdate.UpdateTo(downloadURL, localBinaryPath); err != nil {
		return fmt.Errorf("error replacing binary file: %w", err)
	}
	return nil
}

func getLatestReleaseVersion() (semver.Version, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repoSlug)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Get(url)
	if err != nil {
		return semver.Version{}, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return semver.Version{}, "", fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return semver.Version{}, "", err
	}

	var rel release
	if err := json.Unmarshal(body, &rel); err != nil {
		return semver.Version{}, "", err
	}

	tag := strings.TrimPrefix(rel.TagName, "v")
	version, err := semver.Parse(tag)
	if err != nil {
		return semver.Version{}, "", fmt.Errorf("invalid release tag %q: %w", rel.TagName, err)
	}

	downloadURL, err := pickReleaseAssetURL(rel.Assets)
	if err != nil {
		return semver.Version{}, "", err
	}

	return version, downloadURL, nil
}

func pickReleaseAssetURL(assets []struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}) (string, error) {
	candidates := releaseAssetCandidates()
	for _, candidate := range candidates {
		for _, asset := range assets {
			if asset.Name == candidate {
				return asset.BrowserDownloadURL, nil
			}
		}
	}

	for _, asset := range assets {
		if slices.ContainsFunc(candidates, func(candidate string) bool {
			return asset.Name == candidate || asset.Name == candidate+".gz" || asset.Name == candidate+".zip" || asset.Name == candidate+".tar.gz"
		}) {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no release asset matched runtime %s/%s", runtime.GOOS, runtime.GOARCH)
}

func releaseAssetCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			fmt.Sprintf("gopanel_mac_%s", runtime.GOARCH),
			fmt.Sprintf("gopanel_darwin_%s", runtime.GOARCH),
		}
	case "windows":
		return []string{
			fmt.Sprintf("gopanel_windows_%s.exe", runtime.GOARCH),
			fmt.Sprintf("gopanel_windows_%s", runtime.GOARCH),
		}
	default:
		return []string{
			fmt.Sprintf("gopanel_%s_%s", runtime.GOOS, runtime.GOARCH),
		}
	}
}

func currentVersion() (semver.Version, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return semver.Version{}, false
	}
	versionText := strings.TrimPrefix(info.Main.Version, "v")
	version, err := semver.Parse(versionText)
	if err != nil {
		return semver.Version{}, false
	}
	return version, true
}
