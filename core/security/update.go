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

	osLower := runtime.GOOS
	goArch := runtime.GOARCH
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/gopanel_%s_%s",
		"BapiGso", "gopanel", osLower, goArch)

	return rel.PublishedAt, downloadURL, nil
}

func updateBinaryIfNeeded() error {
	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot get executable path: %w", err)
	}

	// 获取二进制文件的绝对路径
	localBinaryPath, err := filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("cannot get absolute path: %w", err)
	}

	// 获取本地二进制文件的最后修改时间
	localFileInfo, err := os.Stat(localBinaryPath)
	if err != nil {
		return fmt.Errorf("error stating local binary: %w", err)
	}
	localFileModTime := localFileInfo.ModTime()

	// 获取最新发布日期
	latestReleaseDate, downloadURL, err := getLatestReleaseDate()
	if err != nil {
		return fmt.Errorf("error fetching latest release date: %w", err)
	}

	// 比较发布日期与本地二进制修改时间
	if latestReleaseDate.After(localFileModTime) {
		fmt.Println("Newer release found. Downloading updated binary...")

		// 下载新的二进制文件
		resp, err := http.Get(downloadURL)
		if err != nil {
			return fmt.Errorf("error downloading binary: %w", err)
		}
		defer resp.Body.Close()

		// 创建临时文件
		tmpFile, err := os.CreateTemp(filepath.Dir(localBinaryPath), "gopanel.tmp.*")
		if err != nil {
			return fmt.Errorf("error creating temporary file: %w", err)
		}
		tmpPath := tmpFile.Name()

		// 确保在发生错误时删除临时文件
		defer os.Remove(tmpPath)

		// 将下载的内容写入临时文件
		_, err = io.Copy(tmpFile, resp.Body)
		if err != nil {
			tmpFile.Close()
			return fmt.Errorf("error writing to temporary file: %w", err)
		}

		// 关闭临时文件
		if err = tmpFile.Close(); err != nil {
			return fmt.Errorf("error closing temporary file: %w", err)
		}

		// 设置执行权限
		if err = os.Chmod(tmpPath, 0755); err != nil {
			return fmt.Errorf("error setting executable permissions: %w", err)
		}

		// 重命名临时文件以替换原文件
		if err = os.Rename(tmpPath, localBinaryPath); err != nil {
			return fmt.Errorf("error replacing binary file: %w", err)
		}

		fmt.Println("Binary updated successfully.")
	} else {
		return fmt.Errorf("latest release date is %s. Local binary is up-to-date", latestReleaseDate.Format("2006-01-02"))
	}

	return nil
}
