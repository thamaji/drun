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
	"time"

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
		os.Exit(0)
	}

	if version {
		fmt.Fprintln(output, "1.1.0")
		os.Exit(0)
	}

	args := flag.Args()

	if len(args) < 1 {
		usage()
		os.Exit(1)
	}

	image := args[0]

	if len(args) >= 1 {
		args = args[1:]
	}

	workdir, err := filepath.Abs("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	user, err := user.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
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
		"--env", "debian_chroot=drun",
	}

	if hostname, err := os.Hostname(); err == nil {
		run = append(run, "--hostname", hostname)
	}

	if _, err := os.Stat("/etc/localtime"); err == nil {
		run = append(run, "--volume", "/etc/localtime:/etc/localtime:ro")
	} else {
		timezone, _ := time.Now().In(time.Local).Zone()
		run = append(run, "--env", "TZ="+timezone)
	}

	if v, ok := os.LookupEnv("LANG"); ok {
		run = append(run, "--env", "LANG="+v)
	}

	if v, ok := os.LookupEnv("LANGUAGE"); ok {
		run = append(run, "--env", "LANGUAGE="+v)
	}

	if v, ok := os.LookupEnv("LC_ALL"); ok {
		run = append(run, "--env", "LC_ALL="+v)
	}

	if v, ok := os.LookupEnv("DISPLAY"); ok {
		run = append(run, "--env", "DISPLAY="+v)

		path := filepath.Join(user.HomeDir, ".Xauthority")
		if v, ok := os.LookupEnv("XAUTHORITY"); ok {
			path = v
		}
		if _, err := os.Stat(path); err == nil {
			run = append(run, "--volume", path+":/root/.Xauthority:ro")
			run = append(run, "--env", "XAUTHORITY=/root/.Xauthority")
		}
	}

	if v, ok := os.LookupEnv("TERM"); ok {
		run = append(run, "--env", "TERM="+v)
	}

	if v, ok := os.LookupEnv("COLORTERM"); ok {
		run = append(run, "--env", "COLORTERM="+v)
	}

	if terminal.IsTerminal(syscall.Stdin) {
		run = append(run, "--tty")

		if cols, lines, err := terminal.GetSize(syscall.Stdin); err == nil {
			run = append(run, "--env", "COLUMNS="+strconv.Itoa(cols))
			run = append(run, "--env", "LINES="+strconv.Itoa(lines))
		}
	}

	for _, volume := range volumes {
		run = append(run, "--volume", volume+":"+volume+":rw")
	}

	run = append(run, image)
	run = append(run, args...)

	if dry {
		fmt.Fprintln(output, "docker", strings.Join(run, " "))
		os.Exit(0)
	}

	cmd := exec.Command("docker", run...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
