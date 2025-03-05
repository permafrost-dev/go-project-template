package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/blang/semver"
)

type BuildVersionTool struct {
	projectPath string
	version     string
	commit      string
	buildDate   string
}

func getTag(match ...string) (string, *semver.PRVersion) {
	args := append([]string{
		"describe", "--tags",
	}, match...)
	tag, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", nil
	}
	tagParts := strings.Split(string(tag), "-")
	if len(tagParts) == 3 {
		if ahead, err := semver.NewPRVersion(tagParts[1]); err == nil {
			return tagParts[0], &ahead
		}
	} else if len(tagParts) == 4 {
		if ahead, err := semver.NewPRVersion(tagParts[2]); err == nil {
			return tagParts[0] + "-" + tagParts[1], &ahead
		}
	}

	return string(tag), nil
}

func UpdateVersion(version, commit string) {
	buildDate := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
	versionFn := "./internal/version/version.go"
	if _, err := os.ReadFile(versionFn); err != nil {
		fmt.Println(err)
		return
	}

	if strings.EqualFold(version, commit) {
		version = "v0.0.0-dev"
	}

	dataStr := strings.TrimSpace(fmt.Sprintf(`
package version

const Version string = "%s"      // @version
const BuildCommit string = "%s"  // @build-commit
const BuildDate string = "%s" // @build-date
`, strings.TrimSpace(version), strings.TrimSpace(commit), strings.TrimSpace(buildDate)))

	if err := os.WriteFile(versionFn, []byte(dataStr), 0644); err != nil {
		fmt.Println(err)
		return
	}

	exec.Command("go", "fmt", versionFn).Run()
}

func GetCommandOutputString(command string, args ...string) (string, error) {
	out, err := exec.Command(command, args...).Output()

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func main() {

	commit, err := GetCommandOutputString("git", "rev-parse", "--short=10", "HEAD")
	if err != nil {
		commit = "unknown"
	}

	if tags, err := exec.Command("git", "tag").Output(); err != nil || len(tags) == 0 {
		// no tags found -- fetch them
		exec.Command("git", "fetch", "--tags").Run()
	}

	// Find the last vX.X.X Tag and get how many builds we are ahead of it.
	versionStr, ahead := getTag("--match", "v*")
	version, err := semver.ParseTolerant(versionStr)
	if err != nil {
		UpdateVersion(version.String(), commit)
		return
	}
	// Get the tag of the current revision.
	tag, _ := getTag("--exact-match")
	if tag == versionStr {
		UpdateVersion(version.String(), tag)
		return
	}

	// If we don't have any tag assume "dev"
	if tag == "" || strings.HasPrefix(tag, "nightly") {
		tag = "dev"
	}

	// Get the most likely next version:
	if !strings.Contains(version.String(), "rc") {
		version.Patch = version.Patch + 1
	}

	if pr, err := semver.NewPRVersion(tag); err == nil {
		// append the tag as pre-release name
		version.Pre = append(version.Pre, pr)
	}

	if ahead != nil {
		// if we know how many commits we are ahead of the last release, append that too.
		version.Pre = append(version.Pre, *ahead)
	}

	UpdateVersion(version.String(), tag)
}
