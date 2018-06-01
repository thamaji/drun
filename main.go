package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

func usage() {
	output := flag.CommandLine.Output()
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Usage: "+os.Args[0]+" [OPTIONS] IMAGE [COMMAND] [ARG...]")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Run a command in a new container")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Options:")
	flag.CommandLine.PrintDefaults()
}

func main() {
	flag.Usage = usage
	output := flag.CommandLine.Output()

	var dry bool
	var version bool
	var help bool

	flag.BoolVar(&dry, "dry", false, "dry run")
	flag.BoolVar(&version, "v", false, "show version")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()

	if help {
		usage()
		return
	}

	if version {
		fmt.Fprintln(output, "1.0.0")
		return
	}

	args := flag.Args()

	if len(args) < 1 {
		usage()
		return
	}

	image := args[0]

	if len(args) >= 1 {
		args = args[1:]
	}

	workdir, err := filepath.Abs("")
	if err != nil {
		return
	}

	user, err := user.Current()
	if err != nil {
		return
	}

	volumes := []string{workdir}
	for _, arg := range args {
		for path := filepath.Clean(arg); path != ""; path, _ = filepath.Split(filepath.Clean(path)) {
			stat, err := os.Stat(path)
			if err != nil {
				continue
			}

			path, err = filepath.Abs(path)
			if err != nil {
				return
			}

			if !stat.IsDir() {
				path = filepath.Dir(path)
			}

			volumes = append(volumes, path)

			break
		}
	}

	run := []string{
		"run", "--interactive", "--rm", "--network", "host",
		"--user", user.Uid + ":" + user.Gid,
		"--workdir", workdir,
	}

	if terminal.IsTerminal(syscall.Stdin) {
		run = append(run, "--tty")

		if cols, lines, err := terminal.GetSize(syscall.Stdin); err == nil {
			run = append(run, "--env")
			run = append(run, "COLUMNS="+strconv.Itoa(cols))
			run = append(run, "--env")
			run = append(run, "LINES="+strconv.Itoa(lines))
		}
	}

	for _, volume := range volumes {
		run = append(run, "--volume")
		run = append(run, volume+":"+volume+":rw")
	}

	run = append(run, image)
	run = append(run, args...)

	if dry {
		fmt.Fprintln(output, "docker", strings.Join(run, " "))
		return
	}

	cmd := exec.Command("docker", run...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
}
