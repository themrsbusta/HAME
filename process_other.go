//go:build !windows && !linux && !android

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func hideProcess(cmd *exec.Cmd)                           {}
func startOwned(cmd *exec.Cmd) error                      { return cmd.Start() }
func closeOwnedJob()                                      {}
func installConsoleHandler(cancel func()) (func(), error) { return func() {}, nil }
func setConsoleTitle(s string)                            {}
func openBrowser(url string)                              {}
func showConfigurator(path string) error                  { return fmt.Errorf("configurator requires Windows") }

const manualPlatform = true

func shutdownSignals() []os.Signal  { return []os.Signal{os.Interrupt} }
func waitOwned(cmd *exec.Cmd) error { return cmd.Wait() }
func killOwned(cmd *exec.Cmd) error { return cmd.Process.Kill() }
