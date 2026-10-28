//go:build linux || android

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestLinuxOwnedProcessGroup(t *testing.T) {
	shell := "/bin/sh"
	if runtime.GOOS == "android" {
		shell = "/system/bin/sh"
	}
	independent := exec.Command(shell, "-c", "exec sleep 60")
	if err := independent.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { independent.Process.Kill(); independent.Wait() }()
	owned := exec.Command(shell, "-c", "sleep 60 & echo $!; wait")
	pipe, err := owned.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = startOwned(owned); err != nil {
		t.Fatal(err)
	}
	defer killOwned(owned)
	line, err := bufio.NewReader(pipe).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	child, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatal(err)
	}
	if err = killOwned(owned); err != nil {
		t.Fatal(err)
	}
	waitOwned(owned)
	for i := 0; i < 50; i++ {
		data, er := os.ReadFile(fmt.Sprintf("/proc/%d/status", child))
		if os.IsNotExist(er) || strings.Contains(string(data), "State:\tZ") {
			break
		}
		time.Sleep(20 * time.Millisecond)
		if i == 49 {
			t.Fatal("owned descendant survived")
		}
	}
	if err = independent.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatal("independent process killed", err)
	}
}
