package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestExtraStartupArguments(t *testing.T) {
	profile := &Profile{Flags: map[string]bool{"--tools": true, "--threads": true}}
	args, err := validateStartupArgs(`--tools all --threads 8`, profile)
	if err != nil || !reflect.DeepEqual(args, []string{"--tools", "all", "--threads", "8"}) {
		t.Fatal(args, err)
	}
	e := &Engine{cfg: Config{MainModel: "main", CompilerModel: "compiler", GPULayers: "auto"}, extraMain: args}
	if !strings.Contains(strings.Join(e.commonArgs(false), " "), "--tools all") {
		t.Fatal("main args not passed")
	}
	if strings.Contains(strings.Join(e.commonArgs(true), " "), "--tools") {
		t.Fatal("main args leaked to compiler")
	}
}
func TestExtraArgsQuotingAndNoShell(t *testing.T) {
	args, err := parseStartupArgs(`--path-example "C:\models with spaces\file" --tools "all; echo never-executed"`)
	if err != nil || args[1] != `C:\models with spaces\file` || args[3] != "all; echo never-executed" {
		t.Fatal(args, err)
	}
	if _, err = parseStartupArgs(`--tools "all`); err == nil {
		t.Fatal("unclosed quote accepted")
	}
}
func TestExtraArgsCannotOverrideManagedContext(t *testing.T) {
	for _, raw := range []string{`--ctx-size 9999`, `--model=other.gguf`, `-c9999`, `--port 9999`, `--slot-save-path x`} {
		if _, err := validateStartupArgs(raw, nil); err == nil {
			t.Fatal("managed option accepted", raw)
		}
	}
	if _, err := validateStartupArgs(`--not-supported all`, &Profile{Flags: map[string]bool{}}); err == nil {
		t.Fatal("unsupported argument accepted")
	}
}

func TestKnownShortExtraAliasNotConfusedWithContext(t *testing.T){p:=&Profile{Flags:map[string]bool{"-fa":true}};if _,err:=validateStartupArgs("-fa on",p);err!=nil{t.Fatal(err)}}
