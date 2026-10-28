//go:build windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSleepHelper(t *testing.T) {
	if os.Getenv("HAME_SLEEP_HELPER") != "1" {
		t.Skip("subprocess helper")
	}
	time.Sleep(30 * time.Second)
}
func TestJobParentHelper(t *testing.T) {
	if os.Getenv("HAME_JOB_PARENT_HELPER") != "1" {
		t.Skip("subprocess helper")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSleepHelper$")
	cmd.Env = append(os.Environ(), "HAME_SLEEP_HELPER=1")
	if err := startOwned(cmd); err != nil {
		fmt.Println("HELPER_ERROR=" + err.Error())
		os.Exit(1)
	}
	fmt.Printf("OWNED_PID=%d\n", cmd.Process.Pid)
	time.Sleep(30 * time.Second)
}
func TestKillOnlyOwnedProcesses(t *testing.T) {
	unowned := exec.Command(os.Args[0], "-test.run=^TestSleepHelper$")
	unowned.Env = append(os.Environ(), "HAME_SLEEP_HELPER=1")
	hideProcess(unowned)
	if err := unowned.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { unowned.Process.Kill(); unowned.Wait() }()
	parent := exec.Command(os.Args[0], "-test.run=^TestJobParentHelper$")
	parent.Env = append(os.Environ(), "HAME_JOB_PARENT_HELPER=1")
	hideProcess(parent)
	pipe, err := parent.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = parent.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { parent.Process.Kill(); parent.Wait() }()
	pidCh := make(chan uint32, 1)
	go func() {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "OWNED_PID=") {
				pid, _ := strconv.ParseUint(strings.TrimPrefix(line, "OWNED_PID="), 10, 32)
				pidCh <- uint32(pid)
				return
			}
		}
		pidCh <- 0
	}()
	var pid uint32
	select {
	case pid = <-pidCh:
	case <-time.After(8 * time.Second):
		t.Fatal("worker helper timed out")
	}
	if pid == 0 {
		t.Fatal("worker did not start")
	}
	ownedHandle, err := syscall.OpenProcess(0x100000, false, pid)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(ownedHandle)
	unownedHandle, err := syscall.OpenProcess(0x100000, false, uint32(unowned.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(unownedHandle)
	if err = parent.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	parent.Wait()
	state, err := syscall.WaitForSingleObject(ownedHandle, 5000)
	if err != nil || state != 0 {
		t.Fatal("owned worker survived parent", state, err)
	}
	state, err = syscall.WaitForSingleObject(unownedHandle, 0)
	if err != nil || state != 258 {
		t.Fatal("unowned process was terminated", state, err)
	}
}
