package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ReleaseAssetResolver resolves a GitHub release asset URL for the given repo, goos, and goarch.
// Injectable for tests. Returns empty string on failure (best-effort).
type ReleaseAssetResolver func(repo, goos, goarch string) (string, error)

// GitHubReleaseAssetURL resolves the latest release asset URL for a GitHub repo.
// Best-effort: matches goos/goarch in asset names; prefers .tar.gz then .zip.
// Returns empty string on any failure (API error, no matching asset, etc.).
func GitHubReleaseAssetURL(get Downloader) ReleaseAssetResolver {
	return func(repo, goos, goarch string) (string, error) {
		if get == nil {
			get = HTTPGet
		}

		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
		data, err := get(apiURL)
		if err != nil {
			return "", err
		}

		var release struct {
			Assets []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			} `json:"assets"`
		}
		if err := json.Unmarshal(data, &release); err != nil {
			return "", err
		}

		// Build preference list: .tar.gz first, then .zip
		candidates := []string{}
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			// Match goos and goarch in name
			if strings.Contains(name, goos) && strings.Contains(name, goarch) {
				if strings.HasSuffix(name, ".tar.gz") {
					// Prefer tar.gz: prepend
					candidates = append([]string{asset.BrowserDownloadURL}, candidates...)
				} else if strings.HasSuffix(name, ".zip") {
					candidates = append(candidates, asset.BrowserDownloadURL)
				}
			}
		}

		if len(candidates) > 0 {
			return candidates[0], nil
		}

		return "", fmt.Errorf("no matching asset for %s/%s", goos, goarch)
	}
}
