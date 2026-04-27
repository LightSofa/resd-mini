package core

import (
	"bytes"
	"context"
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"os/exec"
	"path/filepath"
	"regexp"
	"resd-mini/core/shared"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type ActionExecResult struct {
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

func matchActionRule(media shared.MediaInfo) (string, ActionRule, bool) {
	if globalConfig == nil || len(globalConfig.ActionRules) == 0 {
		return "", ActionRule{}, false
	}

	candidates := make([]string, 0, 4)
	suffix := strings.ToLower(strings.TrimSpace(media.Suffix))
	if suffix != "" {
		candidates = append(candidates, suffix)
		if strings.HasPrefix(suffix, ".") {
			candidates = append(candidates, strings.TrimPrefix(suffix, "."))
		} else {
			candidates = append(candidates, "."+suffix)
		}
	}
	classify := strings.ToLower(strings.TrimSpace(media.Classify))
	if classify != "" {
		candidates = append(candidates, classify)
	}

	for _, c := range candidates {
		for key, rule := range globalConfig.ActionRules {
			if strings.ToLower(strings.TrimSpace(key)) != c {
				continue
			}
			if !rule.Enabled || strings.TrimSpace(rule.Command) == "" {
				continue
			}
			return key, rule, true
		}
	}
	return "", ActionRule{}, false
}

func buildActionFileName(media shared.MediaInfo) string {
	fileName := shared.Md5(media.Url)
	if v := shared.GetFileNameFromURL(media.Url); v != "" {
		fileName = v
	}

	if media.Description != "" {
		fileName = sanitizeActionFileName(media.Description)
		fileLen := globalConfig.FilenameLen
		if fileLen <= 0 {
			fileLen = 50
		}
		runes := []rune(fileName)
		if len(runes) > fileLen {
			fileName = string(runes[:fileLen])
		}
	}

	if strings.TrimSpace(fileName) == "" {
		fileName = "resource"
	}

	if media.Suffix != "" && !strings.HasSuffix(strings.ToLower(fileName), strings.ToLower(media.Suffix)) {
		fileName += media.Suffix
	}
	return fileName
}

func sanitizeActionFileName(s string) string {
	fileName := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`).ReplaceAllString(s, "")
	fileName = strings.TrimSpace(fileName)
	fileName = strings.TrimRight(fileName, ". ")
	return fileName
}

func buildActionContext(media shared.MediaInfo) map[string]string {
	defaultDir := strings.TrimSpace(globalConfig.SaveDirectory)
	defaultDir = filepath.Clean(defaultDir)

	return map[string]string{
		"%url%":        media.Url,
		"%filename%":   buildActionFileName(media),
		"%defaultDir%": defaultDir,
		"%suffix%":     media.Suffix,
		"%classify%":   media.Classify,
		"%domain%":     media.Domain,
	}
}

func renderActionCommand(tpl string, ctx map[string]string) string {
	result := tpl
	for k, v := range ctx {
		result = strings.ReplaceAll(result, k, v)
	}
	return result
}

func splitCommandLineSimple(s string) ([]string, bool) {
	// Minimal splitter: supports double quotes to keep spaces together.
	// Good enough for common patterns like:
	//   powershell -ExecutionPolicy Bypass -File "a b.ps1" -url "..." -defaultDir "D:\\"
	var out []string
	var cur strings.Builder
	inQuotes := false
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		out = append(out, cur.String())
		cur.Reset()
	}

	for _, r := range s {
		switch {
		case r == '"':
			inQuotes = !inQuotes
		case !inQuotes && unicode.IsSpace(r):
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	if inQuotes {
		return nil, false
	}
	flush()
	return out, true
}

func isPowerShellFileCommand(commandLine string) bool {
	argv, ok := splitCommandLineSimple(strings.TrimSpace(commandLine))
	if !ok || len(argv) == 0 {
		return false
	}
	exe := strings.ToLower(strings.TrimSpace(argv[0]))
	if exe != "powershell" && exe != "powershell.exe" && exe != "pwsh" && exe != "pwsh.exe" {
		return false
	}
	// Only special-case -File execution; keep other shells/builtins on cmd.exe.
	for _, a := range argv[1:] {
		la := strings.ToLower(strings.TrimSpace(a))
		if la == "-file" {
			return true
		}
	}
	return false
}

func decodeWindowsOutput(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if utf8.Valid(b) {
		return string(b)
	}
	// Common on zh-CN Windows: redirected console output is in GBK/CP936.
	if decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), b); err == nil && utf8.Valid(decoded) {
		return string(decoded)
	}
	// Last resort: make it JSON-safe/readable (avoid U+FFFD spam in UI).
	return string(bytes.ToValidUTF8(b, []byte("?")))
}

func executeActionCommand(rule ActionRule, commandLine string) ActionExecResult {
	timeout := rule.TimeoutSec
	if timeout <= 0 {
		timeout = 120
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		if isPowerShellFileCommand(commandLine) {
			argv, ok := splitCommandLineSimple(strings.TrimSpace(commandLine))
			if ok && len(argv) > 0 {
				cmd = exec.CommandContext(ctx, argv[0], argv[1:]...)
			} else {
				cmd = exec.CommandContext(ctx, "cmd", "/C", commandLine)
			}
		} else {
			cmd = exec.CommandContext(ctx, "cmd", "/C", commandLine)
		}
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", commandLine)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	applyCmdSysProcAttr(cmd)

	result := ActionExecResult{
		Command:  commandLine,
		ExitCode: 0,
	}

	err := cmd.Run()
	if runtime.GOOS == "windows" {
		result.Stdout = strings.TrimSpace(decodeWindowsOutput(stdout.Bytes()))
		result.Stderr = strings.TrimSpace(decodeWindowsOutput(stderr.Bytes()))
	} else {
		result.Stdout = strings.TrimSpace(stdout.String())
		result.Stderr = strings.TrimSpace(stderr.String())
	}
	if err == nil {
		return result
	}

	result.ExitCode = -1
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	}
	if errorsText := strings.TrimSpace(err.Error()); errorsText != "" {
		if result.Stderr == "" {
			result.Stderr = errorsText
		} else {
			result.Stderr = result.Stderr + "\n" + errorsText
		}
	}
	if ctx.Err() == context.DeadlineExceeded {
		if result.Stderr == "" {
			result.Stderr = "command timed out"
		} else {
			result.Stderr = result.Stderr + "\ncommand timed out"
		}
	}
	return result
}

func emitActionProgress(media shared.MediaInfo, status, message string, result ActionExecResult) {
	if httpServerOnce == nil {
		return
	}
	httpServerOnce.send("actionProgress", map[string]interface{}{
		"Id":       media.Id,
		"Status":   status,
		"Message":  message,
		"SavePath": media.SavePath,
		"Detail":   result,
	})
}

func runActionRule(media shared.MediaInfo, rule ActionRule, commandLine string) (ActionExecResult, error) {
	if rule.RunAsync {
		go func() {
			emitActionProgress(media, shared.DownloadStatusHandle, "action command started", ActionExecResult{Command: commandLine})
			result := executeActionCommand(rule, commandLine)
			if result.ExitCode == 0 {
				msg := "action command finished"
				if strings.TrimSpace(rule.SuccessTip) != "" {
					msg = strings.TrimSpace(rule.SuccessTip)
				}
				emitActionProgress(media, shared.DownloadStatusDone, msg, result)
				return
			}
			msg := "action command failed"
			if strings.TrimSpace(rule.FailTip) != "" {
				msg = strings.TrimSpace(rule.FailTip)
			} else if line := firstNonEmptyLine(result.Stderr); line != "" {
				msg = fmt.Sprintf("action command failed: %s", line)
			}
			emitActionProgress(media, shared.DownloadStatusError, msg, result)
		}()
		return ActionExecResult{Command: commandLine, ExitCode: 0}, nil
	}

	emitActionProgress(media, shared.DownloadStatusHandle, "action command started", ActionExecResult{Command: commandLine})
	result := executeActionCommand(rule, commandLine)
	if result.ExitCode == 0 {
		msg := "action command finished"
		if strings.TrimSpace(rule.SuccessTip) != "" {
			msg = strings.TrimSpace(rule.SuccessTip)
		}
		emitActionProgress(media, shared.DownloadStatusDone, msg, result)
		return result, nil
	}
	msg := "action command failed"
	if strings.TrimSpace(rule.FailTip) != "" {
		msg = strings.TrimSpace(rule.FailTip)
	} else if line := firstNonEmptyLine(result.Stderr); line != "" {
		msg = fmt.Sprintf("action command failed: %s", line)
	}
	emitActionProgress(media, shared.DownloadStatusError, msg, result)
	return result, fmt.Errorf(msg)
}

func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		t := strings.TrimSpace(line)
		if t != "" {
			return t
		}
	}
	return ""
}
