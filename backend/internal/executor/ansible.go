package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Request struct {
	TaskName         string
	InventoryContent string
	PlaybookContent  string
	CheckOnly        bool
	TimeoutSeconds   int
}

type Result struct {
	JobID    string `json:"jobId"`
	Command  string `json:"command"`
	ExitCode int    `json:"exitCode"`
	Summary  string `json:"summary"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Status   string `json:"status"`
}

type Runner struct {
	BinPath string
	TmpDir  string
}

func (r Runner) Run(req Request) Result {
	jobID := uuid.NewString()
	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = 300
	}

	inventoryPath, playbookPath, prepareErr := r.prepareFiles(jobID, req.InventoryContent, req.PlaybookContent)
	if prepareErr != nil {
		return Result{
			JobID:    jobID,
			ExitCode: 1,
			Summary:  prepareErr.Error(),
			Status:   "failed",
		}
	}

	args := []string{"-i", inventoryPath, playbookPath}
	if req.CheckOnly {
		args = append(args, "--check")
	}
	command := fmt.Sprintf("%s %s", r.BinPath, strings.Join(args, " "))
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.BinPath, args...)
	stdout, runErr := cmd.CombinedOutput()
	exitCode := 0
	if runErr != nil {
		exitCode = 1
	}
	status := "success"
	if exitCode != 0 {
		status = "failed"
	}
	if ctx.Err() == context.DeadlineExceeded {
		status = "timeout"
	}

	summary := "playbook execution completed"
	if status != "success" {
		summary = "playbook execution failed"
	}

	return Result{
		JobID:    jobID,
		Command:  command,
		ExitCode: exitCode,
		Summary:  summary,
		Stdout:   string(stdout),
		Stderr:   errString(runErr),
		Status:   status,
	}
}

func (r Runner) prepareFiles(jobID string, inventoryContent string, playbookContent string) (string, string, error) {
	if inventoryContent == "" {
		inventoryContent = "[all]\n127.0.0.1 ansible_connection=local\n"
	}
	if playbookContent == "" {
		playbookContent = "---\n- hosts: all\n  gather_facts: false\n  tasks:\n    - debug: msg='empty playbook scaffold'\n"
	}
	if err := os.MkdirAll(r.TmpDir, 0o755); err != nil {
		return "", "", err
	}
	inventoryPath := filepath.Join(r.TmpDir, fmt.Sprintf("%s_inventory.ini", jobID))
	playbookPath := filepath.Join(r.TmpDir, fmt.Sprintf("%s_playbook.yml", jobID))
	if err := os.WriteFile(inventoryPath, []byte(inventoryContent), 0o600); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(playbookPath, []byte(playbookContent), 0o600); err != nil {
		return "", "", err
	}
	return inventoryPath, playbookPath, nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
