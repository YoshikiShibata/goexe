// Copyright © 2021 Yoshiki Shibata. All rights reserved.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/YoshikiShibata/tools/util/files"
)

const version = "1.0.2"

var (
	pFlag = flag.Int("cl", 20, "concurrency level")
	vFlag = flag.Bool("v", false, "verbose")
	wFlag = flag.Bool("w", false, "overwrite file by elapsed time")
)

func showUsageAndExit() {
	fmt.Fprintf(os.Stderr, "usage: goexe [-p courrency_level] exec_file\n")
	os.Exit(1)
}

type command struct {
	no          int
	name        string
	args        []string
	err         error
	output      bytes.Buffer
	elapsedTime time.Duration
}

func main() {
	fmt.Printf("goexe version: %s\n", version)

	startTime := time.Now()
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
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		name, args, err := parseCommandLine(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q: %v\n", err)
			os.Exit(1)
		}
		commands = append(commands, &command{name: name, args: args})
	}

	sem := make(chan struct{}, *pFlag)
	var wg sync.WaitGroup

	noOfCommands := len(commands)
	for i, cmd := range commands {
		i := i
		sem <- struct{}{}
		wg.Add(1)
		go func(cmd *command) {
			execCommand(cmd, i+1, noOfCommands)
			<-sem
			wg.Done()
		}(cmd)
	}
	wg.Wait()

	passCount := 0
	failCount := 0
	for _, cmd := range commands {
		if cmd.err == nil {
			passCount++
			continue
		}
		failCount++
	}
	fmt.Printf("Result: %d passed, %d failed, %d total\n",
		passCount, failCount, passCount+failCount)
	fmt.Printf("Elapsed time: %v\n", time.Since(startTime))

	if *wFlag {
		saveByElapsedTime(args[0], commands)
	}

	if failCount > 0 {
		os.Exit(1)
	}
}

func parseCommandLine(line string) (name string, args []string, err error) {
	tokens := strings.Split(line, " ")
	return tokens[0], tokens[1:], nil
}

func execCommand(cmd *command, index, noOfCommands int) {
	fmt.Printf("START: %s %s (%d/%d)\n",
		cmd.name,
		flatenStrings(cmd.args),
		index,
		noOfCommands)
	start := time.Now()

	execCmd := exec.Command(cmd.name, cmd.args...)
	execCmd.Stdout = &cmd.output
	execCmd.Stderr = &cmd.output

	if err := execCmd.Start(); err != nil {
		cmd.err = err
		return
	}

	cmd.err = execCmd.Wait()
	cmd.elapsedTime = time.Since(start)

	if cmd.err == nil {
		fmt.Printf("PASS : %s %s (%v)\n",
			cmd.name, flatenStrings(cmd.args), cmd.elapsedTime)
		if *vFlag {
			fmt.Printf("%s\n\n", cmd.output.String())
		}
		return
	}

	fmt.Printf("FAIL : %s %s\n", cmd.name, flatenStrings(cmd.args))
	fmt.Printf("%s\n", cmd.output.String())
	fmt.Printf("=====: %s %s (%v)\n",
		cmd.name, flatenStrings(cmd.args), cmd.elapsedTime)
}

func flatenStrings(strings []string) string {
	result := strings[0]
	for _, str := range strings[1:] {
		result += " " + str
	}
	return result
}

func saveByElapsedTime(filename string, cmds []*command) {
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[j].elapsedTime < cmds[i].elapsedTime
	})

	_ = os.Remove(filename + ".old") // ignore error

	if err := os.Rename(filename, filename+".old"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to rename %s\n", filename)
		return
	}

	file, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create %s\n", err)
		return
	}
	defer file.Close()

	for _, cmd := range cmds {
		fmt.Fprintln(file,
			fmt.Sprintf("%s %s", cmd.name, flatenStrings(cmd.args)))
	}
}
