package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Version is set at build time via ldflags.
var Version = "dev"

const (
	latestReleaseURL = "https://api.github.com/repos/datagendev/datagen-cli/releases/latest"
	cooldownDuration = 24 * time.Hour
	checkTimeout     = 3 * time.Second
)

type updateCache struct {
	LastChecked   time.Time `json:"last_checked"`
	LatestVersion string    `json:"latest_version"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckForUpdate starts a background goroutine that checks for a newer CLI
// version. Returns a channel that receives a user-facing message (empty if
// no update or on any error).
func CheckForUpdate() <-chan string {
	ch := make(chan string, 1)

	go func() {
		defer func() {
			if len(ch) == 0 {
				ch <- ""
			}
		}()

		if Version == "dev" || isCI() {
			return
		}

		if !shouldCheck() {
			return
		}

		latest, err := FetchLatestVersion()
		if err != nil {
			return
		}

		_ = writeCache(latest)

		if IsNewer(Version, latest) {
			ch <- fmt.Sprintf(
				"\nA newer version of datagen is available: %s (current: %s)\nUpdate with: curl -fsSL https://raw.githubusercontent.com/datagendev/datagen-cli/main/install.sh | sh",
				latest, Version,
			)
		}
	}()

	return ch
}

// FetchLatestVersion queries the GitHub releases API and returns the latest
// tag name (e.g. "v0.3.1").
func FetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: checkTimeout}

	req, err := http.NewRequest("GET", latestReleaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name in response")
	}

	return release.TagName, nil
}

// IsNewer returns true if remote is a higher semver than current.
// Both values may optionally have a "v" prefix.
func IsNewer(current, remote string) bool {
	cur := parseSemver(current)
	rem := parseSemver(remote)
	if cur == nil || rem == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		if rem[i] > cur[i] {
			return true
		}
		if rem[i] < cur[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) []int {
	// Strip "v" prefix and anything after a hyphen (e.g. "v0.3.1-2-gabcdef")
	v = strings.TrimPrefix(v, "v")
	if idx := strings.Index(v, "-"); idx != -1 {
		v = v[:idx]
	}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return nil
	}

	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		nums[i] = n
	}
	return nums
}

func cachePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "datagen", "update-check.json"), nil
}

func shouldCheck() bool {
	c, err := readCache()
	if err != nil {
		return true
	}
	return time.Since(c.LastChecked) > cooldownDuration
}

func readCache() (*updateCache, error) {
	p, err := cachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c updateCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func writeCache(latestVersion string) error {
	p, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(updateCache{
		LastChecked:   time.Now(),
		LatestVersion: latestVersion,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func isCI() bool {
	ciVars := []string{
		"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL",
		"CIRCLECI", "TRAVIS", "TF_BUILD", "CODEBUILD_BUILD_ID",
	}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}
