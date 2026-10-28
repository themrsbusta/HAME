//go:build linux || android

package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

const manualPlatform = true

var processMu sync.Mutex
var processGroups = map[int]bool{}
var processesClosed bool

func hideProcess(cmd *exec.Cmd) {}
func startOwned(cmd *exec.Cmd) error {
	processMu.Lock()
	defer processMu.Unlock()
	if processesClosed {
		return fmt.Errorf("HAME encerrando")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	if err := cmd.Start(); err != nil {
		return err
	}
	processGroups[cmd.Process.Pid] = true
	return nil
}
func waitOwned(cmd *exec.Cmd) error {
	err := cmd.Wait()
	processMu.Lock()
	defer processMu.Unlock()
	// Kill remaining descendants before forgetting this process group.
	if processGroups[cmd.Process.Pid] {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		delete(processGroups, cmd.Process.Pid)
	}
	return err
}
func killOwned(cmd *exec.Cmd) error {
	processMu.Lock()
	defer processMu.Unlock()
	if !processGroups[cmd.Process.Pid] {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
func closeOwnedJob() {
	processMu.Lock()
	defer processMu.Unlock()
	processesClosed = true
	for pid := range processGroups {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
}
func shutdownSignals() []os.Signal                        { return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP} }
func installConsoleHandler(cancel func()) (func(), error) { return func() {}, nil }
func setConsoleTitle(s string)                            {}
func openBrowser(url string)                              { fmt.Println("Abra manualmente no navegador: " + url) }
func showConfigurator(path string) error {
	return fmt.Errorf("Linux/Termux: edite %s; configurador gráfico disponível somente no Windows", path)
}
