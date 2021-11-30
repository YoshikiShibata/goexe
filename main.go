// Copyright © 2021 Yoshiki Shibata. All rights reserved.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/YoshikiShibata/tools/util/files"
)

var (
	pFlag = flag.Int("cl", 20, "concurrency level")
)

func showUsageAndExit() {
	fmt.Fprintf(os.Stderr, "usage: goexec [-p courrency_level] exec_file\n")
	os.Exit(1)
}

type command struct {
	no     int
	name   string
	args   []string
	err    error
	output bytes.Buffer
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		showUsageAndExit()
	}

	lines, err := files.ReadAllLines(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	var commands []*command

	for _, line := range lines {
		name, args, err := parseCommandLine(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q: %v\n", err)
			os.Exit(1)
		}
		commands = append(commands, &command{name: name, args: args})
	}

	sem := make(chan struct{}, *pFlag)
	var wg sync.WaitGroup

	for _, cmd := range commands {
		sem <- struct{}{}
		wg.Add(1)
		go func(cmd *command) {
			execCommand(cmd)
			<-sem
			wg.Done()
		}(cmd)
	}
	wg.Wait()

	/*
		for _, cmd := range commands {
			if cmd.err == nil {
				fmt.Printf("PASS: %s %s\n", cmd.name, cmd.args)
				continue
			}
			fmt.Printf("FAIL: %s %s\n", cmd.name, cmd.args)
			fmt.Printf("%s\n\n", cmd.output.String())
		}
	*/
}

func parseCommandLine(line string) (name string, args []string, err error) {
	tokens := strings.Split(line, " ")
	return tokens[0], tokens[1:], nil
}

func execCommand(cmd *command) {
	fmt.Printf("START: %s %s\n", cmd.name, cmd.args)

	execCmd := exec.Command(cmd.name, cmd.args...)
	execCmd.Stdout = &cmd.output
	execCmd.Stderr = &cmd.output

	if err := execCmd.Start(); err != nil {
		cmd.err = err
		return
	}

	cmd.err = execCmd.Wait()
	if cmd.err != nil {
		fmt.Printf("PASS : %s %s\n", cmd.name, cmd.args)
		return
	}
	fmt.Printf("FAIL : %s %s\n", cmd.name, cmd.args)
	fmt.Printf("%s\n\n", cmd.output.String())
}
