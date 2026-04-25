package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	githubAPIURL    = "https://api.github.com/repos/watchflare-io/watchflare/releases/latest"
	downloadBaseURL = "https://get.watchflare.io"
	httpTimeout     = 30 * time.Second
)

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	TarballURL      string
	ChecksumsURL    string
	TarballName     string
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckForUpdate queries GitHub API and returns update info
func CheckForUpdate(currentVersion string) (*UpdateInfo, error) {
	client := &http.Client{Timeout: httpTimeout}

	req, err := http.NewRequest(http.MethodGet, githubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status from GitHub API: %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	updateAvailable := currentVersion != "dev" && semverGreater(latestVersion, currentVersion)

	tarballName := fmt.Sprintf("watchflare-agent_%s_%s_%s.tar.gz", latestVersion, runtime.GOOS, runtime.GOARCH)
	tag := release.TagName // e.g. "v0.28.0"

	info := &UpdateInfo{
		CurrentVersion:  currentVersion,
		LatestVersion:   latestVersion,
		UpdateAvailable: updateAvailable,
		TarballURL:      fmt.Sprintf("%s/%s/%s", downloadBaseURL, tag, tarballName),
		ChecksumsURL:    fmt.Sprintf("%s/%s/watchflare_checksums.txt", downloadBaseURL, tag),
		TarballName:     tarballName,
	}

	return info, nil
}

// semverGreater returns true if a > b using major.minor.patch comparison.
// Falls back to string equality for non-standard version strings.
func semverGreater(a, b string) bool {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		// Strip any pre-release suffix (e.g. "1-beta")
		p = strings.SplitN(p, "-", 2)[0]
		out[i], _ = strconv.Atoi(p)
	}
	return out
}
