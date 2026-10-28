package main

import (
	"errors"
	"fmt"
	"strings"
)

// Windows-style quoting, converted to argv; never evaluated by cmd/PowerShell.
func parseStartupArgs(raw string) ([]string, error) {
	if len(raw) > 8192 {
		return nil, errors.New("argumentos extras excedem 8192 bytes")
	}
	var result []string
	for i := 0; i < len(raw); {
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == '\r' || raw[i] == '\n') {
			i++
		}
		if i == len(raw) {
			break
		}
		var value strings.Builder
		quoted := false
		started := false
		for i < len(raw) {
			if !quoted && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == '\r' || raw[i] == '\n') {
				break
			}
			slashes := 0
			for i < len(raw) && raw[i] == '\\' {
				slashes++
				i++
			}
			if i < len(raw) && raw[i] == '"' {
				value.WriteString(strings.Repeat("\\", slashes/2))
				if slashes%2 == 1 {
					value.WriteByte('"')
					i++
				} else {
					quoted = !quoted
					i++
				}
				started = true
				continue
			}
			value.WriteString(strings.Repeat("\\", slashes))
			if i == len(raw) {
				break
			}
			value.WriteByte(raw[i])
			started = true
			i++
		}
		if quoted {
			return nil, errors.New("aspas duplas não fechadas nos argumentos extras")
		}
		if started {
			result = append(result, value.String())
		}
	}
	return result, nil
}

var protectedStartupArgs = map[string]bool{}

func init() {
	for _, flag := range strings.Fields("--model -m --model-url -mu --hf-repo -hf -hfr --docker-repo -dr --ctx-size -c --gpu-layers -ngl --n-gpu-layers --device -dev --host --port --parallel -np --api-prefix --api-key --api-key-file --alias -a --slot-save-path --slots --no-slots --cache-ram --cache-idle-slots --no-cache-idle-slots --cache-prompt --no-cache-prompt --file -f --binary-file -bf --prompt -p --system-prompt -sys --system-prompt-file -sysf --output --output-file -o --single-turn -st --n-predict --predict -n --simple-io --display-prompt --no-display-prompt --json-schema -j --json-schema-file -jf --grammar --grammar-file --models-dir --models-preset") {
		protectedStartupArgs[flag] = true
	}
}
func validateStartupArgs(raw string, profile *Profile) ([]string, error) {
	args, err := parseStartupArgs(raw)
	if err != nil {
		return nil, err
	}
	for _, arg := range args {
		flag := strings.SplitN(arg, "=", 2)[0]
		blocked := protectedStartupArgs[flag]
		if strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--") && (profile == nil || !profile.hasStartupFlag(flag)) {
			for _, short := range []string{"-ngl", "-np", "-dev", "-sysf", "-sys", "-jf", "-bf", "-m", "-c", "-p", "-f", "-o", "-n", "-j"} {
				if strings.HasPrefix(flag, short) && len(flag) > len(short) {
					blocked = true
				}
			}
		}
		if blocked {
			return nil, fmt.Errorf("%s é controlado pelo HAME; use os campos próprios ou remova o conflito", flag)
		}
		if profile != nil && strings.HasPrefix(flag, "--") && !profile.Flags[flag] {
			return nil, fmt.Errorf("o llama selecionado não anuncia o argumento extra %s", flag)
		}
	}
	return args, nil
}
