package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func _statusFileExists() bool {
	homeDir := _getHomeDir()
	_, err := os.Stat(homeDir + "/.timer/status")

	return err == nil
}

func _logFileExists(day string) bool {
	homeDir := _getHomeDir()
	_, err := os.Stat(homeDir + "/.timer/logs/" + day)

	return err == nil
}

func _createLogFile(day string) {
	homeDir := _getHomeDir()
	_, err := os.Create(homeDir + "/.timer/logs/" + day)
	check(err)
}

func _readStatusFile() (string, string) {
	homeDir := _getHomeDir()
	data, err := os.ReadFile(homeDir + "/.timer/status")
	check(err)

	statusInfo := strings.Split(string(data[:]), ",")

	return statusInfo[0], strings.TrimSpace(statusInfo[1])
}

func _writeStatusFile(status string) {
	homeDir := _getHomeDir()
	err := os.WriteFile(homeDir+"/.timer/status", []byte(status), 0644)
	check(err)
}

func _removeStatusFile() {
	homeDir := _getHomeDir()
	err := os.Remove(homeDir + "/.timer/status")
	check(err)
}

func _formatDuration(duration time.Duration) string {
	formatted := duration.Round(time.Second).String()
	formatted = strings.Replace(formatted, "h", "h ", 1)
	formatted = strings.Replace(formatted, "m", "m ", 1)

	return formatted
}

func _baseDirExists() bool {
	homeDir := _getHomeDir()
	_, err := os.Stat(homeDir + "/.timer")

	return err == nil
}

func _createBaseDir() {
	homeDir := _getHomeDir()
	err := os.Mkdir(homeDir+"/.timer", 0755)
	check(err)

	err = os.Mkdir(homeDir+"/.timer/logs", 0755)
	check(err)

	defaultConfig := "billable_enable=no"

	err = os.WriteFile(homeDir+"/.timer/config", []byte(defaultConfig), 0644)
	check(err)
}

func _readConfig() {
	homeDir := _getHomeDir()
	configFile, err := os.Open(homeDir + "/.timer/config")
	check(err)

	scanner := bufio.NewScanner(configFile)

	for scanner.Scan() {
		entry := strings.Split(scanner.Text(), "=")

		switch entry[0] {

		case "billable_enable":
			config.billable_enable = entry[1] == "yes"

		case "upstream_service":
			config.upstream_service = entry[1]

		case "url":
			switch config.upstream_service {
			case "jira":
				config.JiraServiceConfig.Url = entry[1]
			case "gitlab":
				config.GitlabServiceConfig.Url = entry[1]
			}

		case "token":
			switch config.upstream_service {
			case "jira":
				config.JiraServiceConfig.Token = entry[1]
			case "gitlab":
				config.GitlabServiceConfig.Token = entry[1]
			}

		case "username":
			switch config.upstream_service {
			case "jira":
				config.JiraServiceConfig.Username = entry[1]
			}

		case "default_gitlab_project_id":
			config.GitlabServiceConfig.DefaultProject = entry[1]

		default:
			continue
		}
	}
	check(scanner.Err())
}

func _configIsComplete() bool {

	// if we have an upstream_service validate else ignore config
	if config.upstream_service != "" {
		switch config.upstream_service {
		case "jira":
			return config.JiraServiceConfig.Url != "" && config.JiraServiceConfig.Username != "" && config.JiraServiceConfig.Token != ""
		case "gitlab":
			return config.GitlabServiceConfig.Url != "" && config.GitlabServiceConfig.Token != ""
		}
	}

	return true
}

func _isGitRepo() bool {
	_, err := os.Stat(".git")

	return err == nil
}

func _getHeadRef() string {
	data, err := os.ReadFile(".git/HEAD")
	check(err)

	return string(data)
}

// @TODO: use $OLDPWD ?
func _getLastWorkingDir() (string, error) {
	homeDir := _getHomeDir()
	path, err := os.ReadFile(homeDir + "/.timer/wd")

	return string(path), err
}

func _setWorkingDir(path string) {
	homeDir := _getHomeDir()
	err := os.WriteFile(homeDir+"/.timer/wd", []byte(path), 0644)
	check(err)
}

func _getHomeDir() string {
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())

	// development mode make .timer dir in cwd
	if isGoRun {
		dir, err := os.Getwd()
		check(err)

		return dir
	} else {
		homeDir, err := os.UserHomeDir()
		check(err)

		return homeDir
	}
}

func PrettyPrint(d any) {
	jd, _ := json.MarshalIndent(d, "", "\t")

	fmt.Println(string(jd))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
