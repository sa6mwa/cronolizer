# cronolizer

A simple command line daemon running one command scheduled using a five field
CRON syntax via [Rob Figueiredo](https://github.com/robfig)'s Golang CRON
implementation available at <https://github.com/robfig/cron>.

## Synopsis

The `cronolize` static binary was developed to schedule background executions
inside long-running containers as a *sidecar process* where a normal sidecar
container pattern or cronjob was not possible. It is a light-weight alternative
to `crond` and also a narrow alternative to AWS Event Rules if you want to
conserve those (now limited to 100 per account without requesting more, but was
50).

`cronolize` can run as a sidecar container (only as PID 1 using the foreground
option) in an AWS ECS/Fargate task (cost is per task, not per container) or K8s
pod very easily without changing permissions/rules (using the primary app's
permissions). CronJobs in K8s can however be fairly easily configured for this
task and would be the preferred way in most cases.

## Usage stories

One usage scenario involved rotating log files and cleaning temp files that
overgrew a long-running container's ephemeral storage (log rotation or temp
removal was not handled by the app in this case). Running a sidecar container
for this purpose was not easily accomplished here within our time-constraints.

Another scenario involved a proprietary docker build solution where the app
suffered from degradation and needed to be restarted frequently. The app was
deployed during container boot (which took some time), why a restart without
cycling the container was much faster (save to resolve the leak in the app). As
the use involved restarting an application inside a container, it was not
possible to run as a sidecar and `cronolize` was much simpler to setup than
installing `cronie` or similar cron daemon onto the image.

## Prerequisites

You need Go to compile and produce the `cronolize` binary and `make` (for
example: `apt-get install build-essential`) if you want to use the `Makefile`
(recommended). Download Go from <https://go.dev/>.

## Building

```console
git clone github.com/sa6mwa/cronolizer
cd cronolizer
make clean
make
```

Binary will be called `cronolize` and will be compiled for the local OS and
architecture. The `Makefile` happily cross-compiles for several OS/architecture
combinations...

```console
# build amd64, arm64, arm6, arm7 and 386 for linux...
make linux

# build darwin/amd64 and darwin/arm64
make darwin

# build amd64 and arm64 for freebsd, netbsd and openbsd...
make freebsd netbsd openbsd
```

## Usage

See <https://pkg.go.dev/github.com/robfig/cron/v3> for reference how to format
the CRON schedule. The internal syntax help covers most usage...

```
$ ./cronolize
Welcome to cronolize 0.1 (C) 2022 SA6MWA https://github.com/sa6mwa/cronolizer

Syntax: ./cronolize [options] cronSpec command

Usage of ./cronolize:
  -fg
        Run cron in the foreground instead of as a background daemon process
  -log string
        Log output from stdout and stderr to this file (default "/dev/null")
  -q    Quiet, don't print the PID message at the end or the log entry in the log file
  -shell string
        Full path to shell used to execute command (default "/bin/sh")
  -shellCommandOption string
        Command option used by the shell, usually -c (default "-c")
  -truncate
        Truncate instead of appending to the log file

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
the copy (child process) will run cron (unless the -fg option is issued) and
block indefinitely until killed.
```

## Author

SA6MWA Michel Blomgren, email: <sa6mwa@gmail.com>
