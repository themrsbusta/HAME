//go:build !windows

package main

import (
	"fmt"
	"os/exec"
)

func hideProcess(cmd *exec.Cmd)                           {}
func startOwned(cmd *exec.Cmd) error                      { return cmd.Start() }
func closeOwnedJob()                                      {}
func installConsoleHandler(cancel func()) (func(), error) { return func() {}, nil }
func setConsoleTitle(s string)                            {}
func openBrowser(url string)                              {}
func showConfigurator(path string) error                  { return fmt.Errorf("configurator requires Windows") }
