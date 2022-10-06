package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var user GithubUser
	err = json.Unmarshal(body, &user)
	if err != nil {
		return "", err
	}

	if user.Name == "" {
		return "", errors.New("No name found")
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

	if len(matches) < 2 {
		return "", errors.New("Could not find github username")
	}

	return matches[1], nil
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

	varMap["project.name.full"] = promptUserForInput("Project name: ", path.Base(projectDir))
	varMap["project.name"] = strings.ReplaceAll(varMap["project.name.full"], " ", "-")
	varMap["project.description"] = promptUserForInput("Project description: ", "")
	varMap["project.author.name"] = promptUserForInput("Your full name: ", githubName)
	varMap["project.author.email"] = promptUserForInput("Your email address: ", githubEmail)
	varMap["project.author.github"] = promptUserForInput("Your github username: ", "")

	vendorUsername, _ := GetGithubVendorUsername()
	varMap["project.vendor.github"] = promptUserForInput("User/org vendor github name: ", vendorUsername)

	vendorName, err := GetGithubUserName(varMap["project.vendor.github"])
	if err != nil {
		vendorName = ""
	}

	varMap["project.vendor.name"] = promptUserForInput("User/org vendor name: ", vendorName)
	varMap["date.year"] = time.Now().Local().Format("2020")

	processDirectoryFiles(projectDir, varMap)

	targetDir := projectDir + "/cmd/" + varMap["project.name"]
	os.MkdirAll(targetDir, 0755)
	os.WriteFile(targetDir+"/main.go", []byte("package main\n\n"), 0644)

	fmt.Println("Done!")
}
