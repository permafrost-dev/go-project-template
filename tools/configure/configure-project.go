package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type GithubUser struct {
	Name string `json:"name"`
}

type GithubOrganization struct {
	Login                   string      `json:"login"`
	ID                      int         `json:"id"`
	NodeID                  string      `json:"node_id"`
	URL                     string      `json:"url"`
	ReposURL                string      `json:"repos_url"`
	EventsURL               string      `json:"events_url"`
	HooksURL                string      `json:"hooks_url"`
	IssuesURL               string      `json:"issues_url"`
	MembersURL              string      `json:"members_url"`
	PublicMembersURL        string      `json:"public_members_url"`
	AvatarURL               string      `json:"avatar_url"`
	Description             string      `json:"description"`
	Name                    string      `json:"name"`
	Company                 interface{} `json:"company"`
	Blog                    string      `json:"blog"`
	Location                string      `json:"location"`
	Email                   string      `json:"email"`
	TwitterUsername         string      `json:"twitter_username"`
	IsVerified              bool        `json:"is_verified"`
	HasOrganizationProjects bool        `json:"has_organization_projects"`
	HasRepositoryProjects   bool        `json:"has_repository_projects"`
	PublicRepos             int         `json:"public_repos"`
	PublicGists             int         `json:"public_gists"`
	Followers               int         `json:"followers"`
	Following               int         `json:"following"`
	HTMLURL                 string      `json:"html_url"`
	CreatedAt               time.Time   `json:"created_at"`
	UpdatedAt               time.Time   `json:"updated_at"`
	Type                    string      `json:"type"`
}

func GetGithubOrganizationName(orgName string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/orgs/%s", orgName)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var org GithubOrganization
	err = json.Unmarshal(body, &org)
	if err != nil {
		return "", err
	}

	if org.Name == "" {
		return "", errors.New("no name found")
	}

	return org.Name, nil
}

func getGithubUsernameFromGitRemote() (string, error) {
	out, err := gitCommand("config remote.origin.url")
	if err != nil {
		return "", err
	}

	remoteUrlParts := strings.Split(strings.Replace(strings.TrimSpace(out), ":", "/", -1), "/")
	return remoteUrlParts[1], nil
}

func searchCommitsForGithubUsername() (string, error) {
	out, err := gitCommand(`config user.name`)
	if err != nil {
		return "", err
	}
	authorName := strings.ToLower(strings.TrimSpace(out))

	out, err = gitCommand(`log --author='@users.noreply.github.com' --pretty='%an:%ae' --reverse`)
	if err != nil {
		return "", err
	}

	committers := strings.Split(out, "\n")
	type committer struct {
		name, email string
	}

	committerList := []committer{}
	for _, line := range committers {
		line = strings.TrimSpace(line)
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		name, email := parts[0], parts[1]
		if strings.Contains(name, "[bot]") {
			continue
		}

		if strings.ToLower(name) == authorName {
			committerList = append(committerList, committer{name, email})
		}
	}

	if len(committerList) == 0 {
		return "", nil
	}

	for _, committer := range committerList {
		fmt.Printf("committer: %s\n", committer)
	}

	return strings.Split(committerList[0].email, "@")[0], nil
}

func guessGithubUsername() (string, error) {
	result, err := searchCommitsForGithubUsername()

	fmt.Println(result)

	if err != nil {
		return "", err
	}

	if result != "" {
		return result, nil
	}

	result, err = getGithubUsernameFromGitRemote()

	fmt.Println(result)
	if err != nil {
		return "", err
	}

	return result, nil
}

func gitCommand(args string) (string, error) {
	cmd := exec.Command("git", strings.Split(args, " ")...)
	out, err := cmd.Output()
	return string(out), err
}

func GetGithubUserName(username string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", username)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var user GithubUser
	err = json.Unmarshal(body, &user)
	if err != nil {
		return "", err
	}

	if user.Name == "" {
		return "", errors.New("no name found")
	}

	return user.Name, nil
}

func GetGithubVendorUsername() (string, error) {
	output, err := exec.Command("git", "remote", "get-url", "origin").Output()

	if err != nil {
		return "", err
	}

	url := strings.Trim(string(output), " \t\r\n")

	re := regexp.MustCompile(`(?i)(?:github\.com[:/])([\w-]+/[\w-]+)`)

	matches := re.FindStringSubmatch(url)
	var result string

	if len(matches) > 1 {
		result = strings.Split(matches[1], "/")[0]
		orgName, err := GetGithubOrganizationName(result)

		if err == nil {
			result = orgName
		}
	} else {
		return "", errors.New("could not find github username")
	}

	return result, nil
}

func promptUserForInput(prompt string, defaultValue string) string {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		if defaultValue != "" {
			fmt.Printf("%s (%s) ", prompt, defaultValue)
		} else {
			fmt.Printf("%s ", prompt)
		}

		scanner.Scan()

		input := strings.TrimSpace(scanner.Text())

		if input == "" && defaultValue != "" {
			return defaultValue
		}

		if input != "" {
			return input
		}
	}
}

func stringInArray(str string, arr []string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}

func processDirectoryFiles(dir string, varMap map[string]string) {
	// get the files in the directory
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println(err)
		return
	}

	ignoreFiles := []string{
		".git",
		".gitattributes",
		".gitignore",
		"configure-project.go",
		"go.sum",
	}

	// loop through the files
	for _, file := range files {
		if stringInArray(strings.ToLower(file.Name()), ignoreFiles) {
			continue
		}

		filePath := dir + "/" + file.Name()

		if file.IsDir() {
			processDirectoryFiles(filePath, varMap)
			continue
		}

		bytes, err := os.ReadFile(filePath)

		if err != nil {
			fmt.Println(err)
			continue
		}

		content := string(bytes)

		for key, value := range varMap {
			if file.Name() == "go.mod" {
				tempKey := strings.ReplaceAll(key, ".", "-")
				content = strings.ReplaceAll(content, "/"+tempKey, "/"+value)
				continue
			}

			key = "{{" + key + "}}"
			content = strings.ReplaceAll(content, key, value)
		}

		if string(bytes) != content {
			fmt.Printf("Updating file: %s\n", filePath)
			os.WriteFile(filePath, []byte(content), 0644)
		}
	}
}

func main() {
	// get the current directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	projectDir, err := filepath.Abs(cwd)

	if err != nil {
		fmt.Println(err)
		return
	}

	varMap := make(map[string]string)

	githubNameBytes, err := exec.Command("git", "config", "--global", "user.name").Output()
	if err != nil {
		githubNameBytes = []byte("")
	}

	githubEmailBytes, err := exec.Command("git", "config", "--global", "user.email").Output()
	if err != nil {
		githubEmailBytes = []byte("")
	}

	githubName := strings.Trim(string(githubNameBytes), " \r\n\t")
	githubEmail := strings.Trim(string(githubEmailBytes), " \r\n\t")
	githubUser, _ := guessGithubUsername()

	varMap["project.name.full"] = promptUserForInput("Project name: ", path.Base(projectDir))
	varMap["project.name"] = strings.ReplaceAll(varMap["project.name.full"], " ", "-")
	varMap["project.description"] = promptUserForInput("Project description: ", "")
	varMap["project.author.name"] = promptUserForInput("Your full name: ", githubName)
	varMap["project.author.email"] = promptUserForInput("Your email address: ", githubEmail)
	varMap["project.author.github"] = promptUserForInput("Your github username: ", githubUser)

	vendorUsername, _ := GetGithubVendorUsername()
	varMap["project.vendor.github"] = promptUserForInput("User/org vendor github name: ", vendorUsername)

	vendorName, _ := GetGithubUserName(varMap["project.vendor.github"])
	varMap["project.vendor.name"] = promptUserForInput("User/org vendor name: ", vendorName)

	varMap["date.year"] = time.Now().Local().Format("2020")

	//processDirectoryFiles(projectDir, varMap)

	for key, value := range varMap {
		fmt.Printf("varMap[%s]: %s\n", key, value)
	}

	// targetDir := projectDir + "/cmd/" + varMap["project.name"]
	// os.MkdirAll(targetDir, 0755)
	// os.WriteFile(targetDir+"/main.go", []byte("package main\n\n"), 0644)

	fmt.Println("Done!")
}
