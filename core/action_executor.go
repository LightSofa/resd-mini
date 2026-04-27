package core

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"resd-mini/core/shared"
	"runtime"
	"strings"
	"time"
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
		fileName = regexp.MustCompile(`[^\w\p{Han}]`).ReplaceAllString(media.Description, "")
		fileLen := globalConfig.FilenameLen
		if fileLen <= 0 {
			fileLen = 30
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

func executeActionCommand(rule ActionRule, commandLine string) ActionExecResult {
	timeout := rule.TimeoutSec
	if timeout <= 0 {
		timeout = 120
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", commandLine)
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
	result.Stdout = strings.TrimSpace(stdout.String())
	result.Stderr = strings.TrimSpace(stderr.String())
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
