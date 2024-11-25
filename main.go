package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"
)

// @TODO: Update for multi integration
type TimerConfig struct {
	billable_enable     bool
	upstream_service    string
	JiraServiceConfig   JiraConfig
	GitlabServiceConfig GitlabConfig
}

type TaskDescription struct {
	JobType     string
	Status      string
	Description string
}

var config = TimerConfig{
	billable_enable: false,
}

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
	case "config":
		PrettyPrint(config)
		os.Exit(0)
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

func printUsage() {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)

	fmt.Fprintln(writer, "usage: timer [command args]\n"+
		"\tstart\t [-at 00:00] [task]\t Start tracking time for a task identifier, may be of an upstream task format or unformatted.\n"+
		"\tstop\t [-at 00:00] \t Stop tracking time.\n"+
		"\tcancel\t\t Cancel tracking time.\n"+
		"\tstatus\t\t Prints time tracking status.\n"+
		"\tlog\t [-f yyyy-mm-dd]\t Print log of the current day or from a specified date."+
		"\tconfig\t\t Print current loaded config.")

	fmt.Fprintln(writer, "Advanced usage:\n"+
		"\tps1\t Output prompt complication.\n"+
		"\tprecmd\t Check current directory and prompt to start time tracking, for use as zsh precommmand function.")

	writer.Flush()
}
