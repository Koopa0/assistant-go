package cli

import (
	"fmt"
	"runtime"
	"time"
)

// Build information that can be set at compile time using ldflags
var (
	// Version is the main version number
	Version = "0.1.0-dev"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// GitBranch is the git branch
	GitBranch = "unknown"

	// BuildTime is when the binary was built
	BuildTime = "unknown"

	// BuildUser is who built the binary
	BuildUser = "unknown"
)

// Info contains all version and build information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	GitBranch string `json:"git_branch"`
	BuildTime string `json:"build_time"`
	BuildUser string `json:"build_user"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get returns the complete version information
func Get() *Info {
	return &Info{
		Version:   Version,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		BuildTime: BuildTime,
		BuildUser: BuildUser,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// GetVersion returns just the version string
func GetVersion() string {
	return Version
}

// GetFullVersion returns a detailed version string
func GetFullVersion() string {
	info := Get()
	if GitCommit != "unknown" {
		return fmt.Sprintf("%s (commit: %s, built: %s)", info.Version, GitCommit[:8], BuildTime)
	}
	return info.Version
}

// PrintVersion prints version information in a user-friendly format
func PrintVersion(appName string) {
	info := Get()
	fmt.Printf("%s %s\n", appName, info.Version)

	if GitCommit != "unknown" {
		fmt.Printf("  Git commit: %s\n", GitCommit)
	}
	if GitBranch != "unknown" {
		fmt.Printf("  Git branch: %s\n", GitBranch)
	}
	if BuildTime != "unknown" {
		if buildTime, err := time.Parse(time.RFC3339, BuildTime); err == nil {
			fmt.Printf("  Built: %s\n", buildTime.Format("2006-01-02 15:04:05 MST"))
		} else {
			fmt.Printf("  Built: %s\n", BuildTime)
		}
	}
	if BuildUser != "unknown" {
		fmt.Printf("  Built by: %s\n", BuildUser)
	}
	fmt.Printf("  Go version: %s\n", info.GoVersion)
	fmt.Printf("  Platform: %s\n", info.Platform)
}

// GetBuildInfo returns build information as a map for structured logging
func GetBuildInfo() map[string]interface{} {
	info := Get()
	return map[string]interface{}{
		"version":    info.Version,
		"git_commit": info.GitCommit,
		"git_branch": info.GitBranch,
		"build_time": info.BuildTime,
		"build_user": info.BuildUser,
		"go_version": info.GoVersion,
		"platform":   info.Platform,
	}
}
