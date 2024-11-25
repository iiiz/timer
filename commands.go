package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/AlecAivazis/survey/v2"
)

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

		// @TODO dynamic survey options, allow entry of new value
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
		}

		if config.billable_enable {
			taskSurvey = append(taskSurvey, &survey.Question{
				Name: "Status",
				Prompt: &survey.Select{
					Message: "Status:",
					Options: []string{"Billable", "Not Billable"},
					Default: "Billable",
				},
			})
		}

		taskSurvey = append(taskSurvey, &survey.Question{
			Name:   "Description",
			Prompt: &survey.Input{Message: "Description:"},
		})

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

		if config.upstream_service != "" {
			if config.upstream_service == "gitlab" {
				if isGitlabTaskFormat(task) {
					var didSubmitLog bool

					if checkAndLoadGitlabIssue(task) {
						var seconds int64 = endTime.Sub(startTime).Milliseconds() / 1000

						didSubmitLog = submitGitlabTimeSpent(taskInfo, seconds)
					}

					if didSubmitLog != true {
						fmt.Println(fmt.Sprintf("Warning: %s looks like a gitlab issue branch, this issue is either not found or an error occured. A gitlab worklog was not created for this time period.", task))
					}
				}
			}

			if config.upstream_service == "jira" {
				if isJiraTaskFormat(task) {
					var didSubmitLog bool

					if _checkAndLoadJiraIssue(task) {
						var seconds int64 = endTime.Sub(startTime).Milliseconds() / 1000

						didSubmitLog = _submitJiraWorkLog(jiraCurrentTask.Key, taskInfo, seconds)
					}

					if didSubmitLog != true {
						fmt.Println(fmt.Sprintf("Warning: %s looks like a jira task identifier, this is either not found or an error occured. A jira worklog was not created for this time period.", task))
					}
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
