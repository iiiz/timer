package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/AlecAivazis/survey/v2"
)

type TimerConfig struct {
	upstream_service string
	jira_url         string
	username         string
	access_token     string
}

type JiraIssue struct {
	Key    string          `json:key`
	Fields JiraIssueFields `json:fields`
}

type JiraIssueFields struct {
	Summary      string                `json:summary`
	Description  string                `json:description`
	Created      string                `json:created`
	TimeTracking JiraIssueTimeTracking `json:timetracking`
}

type JiraIssueTimeTracking struct {
	OriginalEstimateSeconds  int `json:originalEstimateSeconds`
	RemainingEstimateSeconds int `json:remainingEstimateSeconds`
	TimeSpentSeconds         int `json:timeSpentSeconds`
}

type TaskDescription struct {
	JobType     string
	Status      string
	Description string
}

var config TimerConfig
var httpClient *http.Client
var currentIssue JiraIssue

/**
 * Main
 * Primary commands and arguments defined here.
 */
func main() {
	if _baseDirExists() != true {
		_createBaseDir()
	}

	_readConfig()

	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	startAtTime := startCmd.String("at", "", "at")

	stopCmd := flag.NewFlagSet("stop", flag.ExitOnError)
	stopAtTime := stopCmd.String("at", "", "at")

	logCmd := flag.NewFlagSet("log", flag.ExitOnError)
	fromDate := logCmd.String("f", "", "f")
	toDate := logCmd.String("t", time.Now().Format("2006-01-02"), "t")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "start":
		startCmd.Parse(os.Args[2:])

		if len(startCmd.Args()) > 0 {
			start(startCmd.Args()[0], *startAtTime)
		} else {
			fmt.Println("No task name provided.")
			os.Exit(1)
		}
	case "status":
		status()
	case "stop":
		stopCmd.Parse(os.Args[2:])

		stop(*stopAtTime)
	case "cancel":
		cancel()
	case "log":
		logCmd.Parse(os.Args[2:])
		if *fromDate != "" {
			logFromTo(*fromDate, *toDate)
		} else {
			logDay(time.Now())
		}
	case "ps1":
		ps1Complication()
	case "precmd":
		preCmd()
	case "help":
		printUsage()
		os.Exit(0)
	default:
		fmt.Println("Unexpected arguments, received ", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

/**
 * Status
 * Read from the status file and output the current task and time difference if existing.
 */
func status() {
	if _statusFileExists() {
		task, startTimeString := _readStatusFile()
		startTime, err := time.Parse(time.RFC3339, startTimeString)
		check(err)

		formattedDiff := _formatDuration(time.Since(startTime))

		fmt.Println("Task", task, "started", formattedDiff, "ago.", startTime.Format(time.ANSIC))

	} else {
		fmt.Println("No task currently started")
	}
	os.Exit(0)
}

/**
 * Start
 * Start a task timer, writes to status file. If status is pre-existing error.
 */
func start(task, atTime string) {
	if _statusFileExists() {
		fmt.Println("Error: A task is already started.")
		os.Exit(1)
	} else {
		now := time.Now()
		var startTime time.Time

		if atTime != "" {
			loc, err := time.LoadLocation("Local")
			check(err)
			startTime, err = time.ParseInLocation("2006-01-02T15:04:05", fmt.Sprintf("%d-%02d-%02dT%s:00", now.Year(), int(now.Month()), now.Day(), atTime), loc)
			check(err)

		} else {
			startTime = now
		}

		if now.Before(startTime) {
			fmt.Println("Error, cannot start task in the future.")
			os.Exit(1)
		}

		if task == "" {
			printUsage()
			os.Exit(1)
		}

		_writeStatusFile(fmt.Sprintf("%s,%s", task, startTime.Format(time.RFC3339)))
		fmt.Println(fmt.Sprintf("Started %s at %s", task, startTime.Format(time.Kitchen)))
	}
}

/**
 * Stop
 * Stop a task timer and commit the time elapsed to the log file.
 */
func stop(atTime string) {
	if _statusFileExists() {
		now := time.Now()
		var endTime time.Time

		if atTime != "" {
			loc, err := time.LoadLocation("Local")
			check(err)
			endTime, err = time.ParseInLocation("2006-01-02T15:04:05", fmt.Sprintf("%d-%02d-%02dT%s:00", now.Year(), int(now.Month()), now.Day(), atTime), loc)
			check(err)

		} else {
			endTime = now
		}

		task, startTimeString := _readStatusFile()
		startTime, err := time.Parse(time.RFC3339, startTimeString)
		check(err)

		formattedDuration := _formatDuration(endTime.Sub(startTime))
		day := startTime.Format("2006-01-02")

		if _logFileExists(day) == false {
			_createLogFile(day)
		}

		fmt.Println(fmt.Sprintf("Stopping %s...", task))

		var taskInfo TaskDescription
		var taskSurvey = []*survey.Question{
			{
				Name: "JobType",
				Prompt: &survey.Select{
					Message: "JobType:",
					Options: []string{
						"Frontend Development",
						"Code Review",
						"Deployment",
						"Internal Meeting",
						"Backend Development",
						"Design",
						"Client Meeting",
						"Quality Assurance",
						"Project Discovery",
						"Project Management",
						"Strategy",
						"Site Analysis",
						"Research",
					},
					Default: "Frontend Development",
				},
			},
			{
				Name: "Status",
				Prompt: &survey.Select{
					Message: "Status:",
					Options: []string{"Billable", "Not Billable"},
					Default: "Billable",
				},
			},
			{
				Name:   "Description",
				Prompt: &survey.Input{Message: "Description:"},
			},
		}

		err = survey.Ask(taskSurvey, &taskInfo)
		check(err)

		homeDir := _getHomeDir()
		logFile, err := os.OpenFile(homeDir+"/.timer/logs/"+day, os.O_APPEND|os.O_WRONLY, 0644)
		check(err)
		defer logFile.Close()

		_, err = logFile.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n", task, formattedDuration, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), base64.StdEncoding.EncodeToString([]byte(taskInfo.Description))))
		check(err)

		_removeStatusFile()

		fmt.Println(fmt.Sprintf("Stopped %s %s elapsed.", task, formattedDuration))

    // check config for a target upstream.
    if ( config.upstream_service != "" && config.upstream_service == "jira" ) {
      if _isTaskFormat(task) {
        var didSubmitLog bool

        if _checkAndLoadJiraIssue(task) {
          var seconds int64 = endTime.Sub(startTime).Milliseconds() / 1000

          didSubmitLog = _submitJiraWorkLog(currentIssue.Key, taskInfo, seconds)
        }

        if didSubmitLog != true {
          fmt.Println(fmt.Sprintf("Warning: %s looks like a jira task identifier, this is either not found or an error occured. A jira worklog was not created for this time period.", task))
        }
      }
    }

	} else {
		fmt.Println("No task started.")
	}
	os.Exit(0)
}

/**
 * Cancel
 * Stops the task timer, removes the status file if it exists.
 */
func cancel() {
	if _statusFileExists() {
		_removeStatusFile()
	} else {
		fmt.Println("No task started.")
	}
	os.Exit(0)
}

func preCmd() {
	cwd, err := os.Getwd()
	check(err)
	pwd := _getLastWorkingDir()

	if !_statusFileExists() {
		// no current status, check the cwd and see if it is a git dir.
		isGit := _isGitRepo()

		if isGit && cwd != pwd {
			head := _getHeadRef()
			branchLeader := strings.Split(head, "/")[2]
			isPossibleTaskIdent := _isTaskFormat(branchLeader)

			reassign := !isPossibleTaskIdent
			reader := bufio.NewReader(os.Stdin)
			var answer string = ""
			var taskIdent string = ""

			if isPossibleTaskIdent {
				fmt.Print(fmt.Sprintf("timer: Git repository detected, would you like to start tracking time with %s as the task id?\n (y/n/o): ", branchLeader))
			} else {
				fmt.Print("timer: Git repository detected, would you like to start tracking time?\n (y/n): ")
			}
			answer, _ = reader.ReadString('\n')
			answer = strings.Replace(answer, "\n", "", -1)

			switch answer {
			case "y":
				if !reassign {
					taskIdent = branchLeader
				}
			case "n":
				_setWorkingDir(cwd)
				os.Exit(0)
			case "o":
				reassign = true
			default:
				_setWorkingDir(cwd)
				fmt.Println("Unknown response, exiting.")
				os.Exit(0)
			}

			if reassign {
				fmt.Print("Enter a task identifier: ")
				newIdent, err := reader.ReadString('\n')
				check(err)

				taskIdent = strings.Replace(newIdent, "\n", "", -1)
			}

			start(taskIdent, "")
		}
	}

	_setWorkingDir(cwd)
}

func ps1Complication() {
	if _statusFileExists() {
		task, startTimeString := _readStatusFile()
		startTime, err := time.Parse(time.RFC3339, startTimeString)
		check(err)

		formattedDiff := _formatDuration(time.Since(startTime))

		fmt.Print(fmt.Sprintf("%s %s", task, formattedDiff))
	} else {
		fmt.Print("<No task>")
	}
}

func logDay(dateTime time.Time) {
	day := dateTime.Format("2006-01-02")

	fmt.Println(dateTime.Format("January 2, 2006"))

	if _logFileExists(day) {
		homeDir := _getHomeDir()
		file, err := os.Open(homeDir + "/.timer/logs/" + day)
		check(err)
		defer file.Close()

		scanner := bufio.NewScanner(file)
		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
		var dayTotalMs int64 = 0

		for scanner.Scan() {
			log := strings.Split(scanner.Text(), ",")

			startTime, err := time.Parse(time.RFC3339, log[2])
			check(err)

			endTime, err := time.Parse(time.RFC3339, log[3])
			check(err)

			dayTotalMs += endTime.Sub(startTime).Milliseconds()

			endFormat := "15:04"

			y1, m1, d1 := startTime.Date()
			y2, m2, d2 := endTime.Date()

			if y1 != y2 || m1 != m2 || d1 != d2 {
				endFormat = "15:04 2006-01-02"
			}

			description, _ := base64.StdEncoding.DecodeString(log[4])

			fmt.Fprintln(writer, fmt.Sprintf("\t%s\t%s\t%s to %s\t%s", log[0], log[1], startTime.Format("15:04"), endTime.Format(endFormat), description))
		}
		var totalDuration time.Duration = time.Duration(dayTotalMs) * time.Millisecond
		fmt.Fprintln(writer, "\tTotal:", _formatDuration(totalDuration))

		writer.Flush()

		check(scanner.Err())

	} else {
		fmt.Println("\t-")
	}
}

func logFromTo(from, to string) {
	fromDate, err := time.Parse("2006-01-02", from)
	check(err)

	toDate, err := time.Parse("2006-01-02", to)
	check(err)

	if toDate.Before(fromDate) {
		fmt.Println(fmt.Sprintf("Error, cannot log from %s to %s", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02")))
		os.Exit(1)
	}

	for {
		logDay(fromDate)

		if !fromDate.Before(toDate) {
			break
		}
		fromDate = fromDate.Add(time.Hour * 24)
	}
}

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

func _isTaskFormat(identifier string) bool {
	taskFmt := regexp.MustCompile(`(?i)[A-Z0-9]+-[0-9]+`)

	return taskFmt.Match([]byte(identifier))
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

	_, err = os.Create(homeDir + "/.timer/config")
	check(err)
}

func _readConfig() {
	homeDir := _getHomeDir()
	configFile, err := os.Open(homeDir + "/.timer/config")
	check(err)

	scanner := bufio.NewScanner(configFile)

  var upstream string = ""
	var url string = ""
	var token string = ""
	var user string = ""

	for scanner.Scan() {
		entry := strings.Split(scanner.Text(), "=")

		switch entry[0] {
    case "upstream_service":
      upstream = entry[1]
		case "jira_url":
			url = entry[1]
		case "access_token":
			token = entry[1]
		case "username":
			user = entry[1]
		default:
			continue
		}
	}
	check(scanner.Err())

	config = TimerConfig{upstream, url, user, token}
}

func _configIsComplete() bool {
	return config.jira_url != "" && config.username != "" && config.access_token != ""
}

func _checkAndLoadJiraIssue(taskKey string) bool {
	if _configIsComplete() {
		if httpClient == nil {
			httpClient = &http.Client{
				CheckRedirect: redirectPolicyFunc,
			}
		}

		req, err := http.NewRequest("GET", config.jira_url+"/issue/"+taskKey, nil)
		check(err)

		auth := base64.StdEncoding.EncodeToString([]byte(config.username + ":" + config.access_token))
		req.Header.Add("Authorization", "Basic "+auth)

		response, err := httpClient.Do(req)
		check(err)

		defer response.Body.Close()

		if response.StatusCode == 200 {
			json.NewDecoder(response.Body).Decode(&currentIssue)
		} else if response.StatusCode == 401 {
			fmt.Println("Warning: Unable to update Jira, please check your configuration.")
		}

		return response.StatusCode == 200
	} else {
		fmt.Println("Warning: Unable to update Jira, please check your configuration.")

		return false
	}
}

func _submitJiraWorkLog(taskKey string, info TaskDescription, seconds int64) bool {
	if _configIsComplete() {
		if httpClient == nil {
			httpClient = &http.Client{
				CheckRedirect: redirectPolicyFunc,
			}
		}

		body, err := json.Marshal(map[string]interface{}{
			"comment":          fmt.Sprintf("Job Type: %s\nStatus: %s\nDescription: %s", info.JobType, info.Status, info.Description),
			"timeSpentSeconds": seconds,
		})
		check(err)

		req, err := http.NewRequest("POST", config.jira_url+"/issue/"+taskKey+"/worklog", bytes.NewBuffer(body))
		req.Header.Set("Content-type", "application/json")
		check(err)

		auth := base64.StdEncoding.EncodeToString([]byte(config.username + ":" + config.access_token))
		req.Header.Add("Authorization", "Basic "+auth)

		response, err := httpClient.Do(req)
		check(err)

		defer response.Body.Close()

		return response.StatusCode == 201
	}

	return false
}

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	auth := base64.StdEncoding.EncodeToString([]byte(config.username + ":" + config.access_token))
	req.Header.Add("Authorization", "Basic "+auth)

	return nil
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

func _getLastWorkingDir() string {
	homeDir := _getHomeDir()
	path, err := os.ReadFile(homeDir + "/.timer/wd")
	check(err)

	return string(path)
}

func _setWorkingDir(path string) {
	homeDir := _getHomeDir()
	err := os.WriteFile(homeDir+"/.timer/wd", []byte(path), 0644)
	check(err)
}

func _getHomeDir() string {
	homeDir, err := os.UserHomeDir()
	check(err)

	return homeDir
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func printUsage() {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)

	fmt.Fprintln(writer, "usage: timer [command args]\n"+
		"\tstart\t [-at 00:00] [task]\t Start tracking time for a task identifier, may be of an upstream task format or unformatted.\n"+
		"\tstop\t\t Stop tracking time.\n"+
		"\tcancel\t\t Cancel tracking time.\n"+
		"\tstatus\t\t Prints time tracking status.\n"+
		"\tlog\t [-f yyyy-mm-dd]\t Print log of the current day or from a specified date.")

	fmt.Fprintln(writer, "Advanced usage:\n"+
		"\tps1\t Output prompt complication.\n"+
		"\tprecmd\t Check current directory and prompt to start time tracking, for use as zsh precommmand function.")

	writer.Flush()
}
