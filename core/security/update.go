package security

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type release struct {
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getLatestReleaseDate(repoOwner, repoName string) (time.Time, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(url)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return time.Time{}, err
	}

	var release release
	if err := json.Unmarshal(body, &release); err != nil {
		return time.Time{}, err
	}

	return release.PublishedAt, nil
}
