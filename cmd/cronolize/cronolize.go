package main

// Cronolizer (C) 2022 SA6MWA https://github.com/sa6mwa/cronolizer
//
// Being rusty in C and unable to find a very simple to use libcron
// implementation in plain C, I went for the Golang quick-fix.
//
// Go does not support fork() very well which is unfortunate if you would like
// your program to run in the background without using nohup or similar
// approaches. The approach utilized in Cronolize is to have the first instance
// execute itself using *(exec.Cmd).Start() with a unique environment variable
// indicating which part of the code the program should run
// (validator/runner-of-itself vs a cron instance blocking forever).

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	version string = "0"
)

const (
	cronolizerEnvVar    string = "__CRONOLIZER__"
	envVarValueExpected string = "INSTANTIATED"
	logFlag             string = "log"
	foregroundFlag      string = "fg"
	helpMsg             string = `
cronSpec is a five field CRON expression. See below or refer to
https://pkg.go.dev/github.com/robfig/cron/v3 for details.

command is the command string to execute via /bin/sh -c (by default). See -h
for more information.

Examples:
cronolize -log /var/log/minute.log "* * * * *" 'date ; echo Hello world'
cronolize -shell /bin/bash "@hourly" 'echo "Last run on $(date)" > /var/opt/output'
cronolize -log out.log "CRON_TZ=Europe/Stockholm 37 13 * * *" 'touch /var/opt/touchable ; echo "Touched file at $(date)"'
cronolize -log /var/logs/nightlyRestart.log "@daily" "echo \"Restarting myservice\" ; supervisorctl restart myservice"

Cron format:

Field name   | Mandatory? | Allowed values  | Allowed special characters
----------   | ---------- | --------------  | --------------------------
Minutes      | Yes        | 0-59            | * / , -
Hours        | Yes        | 0-23            | * / , -
Day of month | Yes        | 1-31            | * / , - ?
Month        | Yes        | 1-12 or JAN-DEC | * / , -
Day of week  | Yes        | 0-6 or SUN-SAT  | * / , - ?

Predefined schedules:

Entry                  | Description                                | Equivalent To
-----                  | -----------                                | -------------
@yearly (or @annually) | Run once a year, midnight, Jan. 1st        | 0 0 1 1 *
@monthly               | Run once a month, midnight, first of month | 0 0 1 * *
@weekly                | Run once a week, midnight between Sat/Sun  | 0 0 * * 0
@daily (or @midnight)  | Run once a day, midnight                   | 0 0 * * *
@hourly                | Run once an hour, beginning of hour        | 0 * * * *

The parent process will start a copy of itself in the background and exit while
the copy (child process) will run cron and block indefinitely until killed
(unless the -fg option is issued).
`
)

// fatal() sends a message to stderr prepended with "Error:" and terminates with
// exit code 1 (fatalf() does not prepend any text, works like log.Fatalf).
func fatal(a ...any) {
	prepend := "Error:"
	// any or interface{} is the question...
	a = append([]interface{}{prepend}, a...)
	fmt.Fprintln(os.Stderr, a...)
	os.Exit(1)
}

func fatalf(format string, a ...any) {
	if len(a) < 1 {
		fatal(format)
	} else {
		fmt.Fprintln(os.Stderr, fmt.Sprintf(format, a...))
		os.Exit(1)
	}
}

func p(format string, a ...any) {
	if len(a) < 1 {
		fmt.Println(format)
	} else {
		fmt.Println(fmt.Sprintf(format, a...))
	}
}

func pe(format string, a ...any) {
	if len(a) < 1 {
		fmt.Fprintln(os.Stderr, format)
	} else {
		fmt.Fprintln(os.Stderr, fmt.Sprintf(format, a...))
	}
}

func main() {
	var isCronProcess bool

	stage, hasEnvVar := os.LookupEnv(cronolizerEnvVar)
	if hasEnvVar && stage == envVarValueExpected {
		isCronProcess = true
	} else {
		isCronProcess = false
	}
	if hasEnvVar {
		os.Unsetenv(cronolizerEnvVar)
	}

	logfile := flag.String(logFlag, os.DevNull, "Log output from stdout and stderr to this file")
	shell := flag.String("shell", "/bin/sh", "Full path to shell used to execute command")
	shellCommandOption := flag.String("shellCommandOption", "-c", "Command option used by the shell, usually -c")
	truncateLog := flag.Bool("truncate", false, "Truncate instead of appending to the log file")
	quiet := flag.Bool("q", false, "Quiet, don't print the PID message at the end or the log entry in the output file")
	foreground := flag.Bool(foregroundFlag, false, "Run cron in the foreground instead of as a background daemon process")

	flag.Parse()

	if len(flag.Args()) != 2 {
		pe("Welcome to cronolize %s (C) 2022 SA6MWA https://github.com/sa6mwa/cronolizer", version)
		pe("")
		pe("Syntax: %s [options] cronSpec command", os.Args[0])
		pe("")
		flag.Usage()
		pe(helpMsg)
		os.Exit(1)
	}

	hasLogFlag := false
	hasForegroundFlag := false
	flag.CommandLine.Visit(func(f *flag.Flag) {
		switch f.Name {
		case logFlag:
			hasLogFlag = true
		case foregroundFlag:
			hasForegroundFlag = true
		}
	})

	if hasLogFlag && hasForegroundFlag {
		fatalf("Syntax error: you can not combine the -%s and the -%s option.", logFlag, foregroundFlag)
	}

	if !*foreground {
		cleanedPath := filepath.Clean(*logfile)
		evaluatedPath, err := filepath.EvalSymlinks(cleanedPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				fatal(err)
			} else {
				evaluatedPath = cleanedPath
			}
		}
		logfile = &evaluatedPath

		var flags int
		if *truncateLog {
			flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND | os.O_TRUNC
		} else {
			flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		}
		logfileFD, err := os.OpenFile(*logfile, flags, 0666)
		if err != nil {
			fatal(err)
		}

		if isCronProcess {
			os.Stdout = logfileFD
			os.Stderr = logfileFD
			log.SetOutput(logfileFD)
		} else {
			logfileFD.Close()
		}
	}

	c := cron.New()
	_, err := c.AddFunc(flag.Args()[0], func() {
		var cmd *exec.Cmd
		if len(*shellCommandOption) != 0 {
			if !*quiet {
				log.Printf("Running: %s", strings.Join([]string{*shell, *shellCommandOption, flag.Args()[1]}, " "))
			}
			cmd = exec.Command(*shell, *shellCommandOption, flag.Args()[1])
		} else {
			if !*quiet {
				log.Printf("Running: %s", strings.Join([]string{*shell, flag.Args()[1]}, " "))
			}
			cmd = exec.Command(*shell, flag.Args()[1])
		}
		if !*foreground {
			cmd.Stdin = os.Stdin
		} else {
			cmd.Stdin = nil
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fatal(err)
		}
	})
	if err != nil {
		fatal(err)
	}

	if isCronProcess || *foreground {
		// Start cron and wait forever.
		c.Start()
		for {
			time.Sleep(time.Duration(math.MaxInt64))
		}
	}

	// Set the environment variable that signal the next execution to start cron
	// and wait forever instead of executing itself.
	err = os.Setenv(cronolizerEnvVar, envVarValueExpected)
	if err != nil {
		fatal(err)
	}

	// Run myself again with the same arguments, but with the added environment variable.
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	err = cmd.Start()
	if err != nil {
		fatal(err)
	}
	if !*quiet {
		p("Running cron job as PID %d", cmd.Process.Pid)
	}
}
