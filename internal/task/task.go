// Package task provides a task manager for CLI output with progress spinners.
package task

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chelnak/ysmrr"
)

// Process exit codes for a task error.
const (
	// exitFailure is the exit code for a fatal task error.
	exitFailure = 1
	// exitInterrupted is the exit code for a task that an interrupt ended, as shells report SIGINT.
	exitInterrupted = 130
)

// Manager manages tasks with spinners for CLI output.
type Manager struct {
	sm ysmrr.SpinnerManager
	// stderr receives fatal errors when no spinner shows them.
	stderr io.Writer
	// exit ends the process after a fatal error.
	exit       func(code int)
	isOut      bool
	noProgress bool
}

// Task represents a single task with optional spinner.
type Task struct {
	spinner *ysmrr.Spinner
	manager *Manager
	title   string
}

// NewManager creates a new task manager.
func NewManager(jsonOutput, unixOutput bool) *Manager {
	isOut := !jsonOutput || unixOutput

	manager := &Manager{
		sm:         ysmrr.NewSpinnerManager(),
		stderr:     os.Stderr,
		exit:       os.Exit,
		isOut:      isOut,
		noProgress: unixOutput,
	}
	if isOut && !unixOutput {
		manager.sm.Start()
	}

	return manager
}

// Reset resets the spinner manager.
func (tm *Manager) Reset() {
	if tm.isOut && !tm.noProgress {
		tm.sm.Stop()
		tm.sm = ysmrr.NewSpinnerManager()
		tm.sm.Start()
	}
}

// Stop stops the spinner manager.
func (tm *Manager) Stop() {
	if tm.isOut && !tm.noProgress {
		tm.sm.Stop()
	}
}

// Println prints a message.
func (tm *Manager) Println(message string) {
	if tm.noProgress {
		_, _ = fmt.Fprintln(os.Stdout, message)

		return
	}

	if tm.isOut {
		context := &Task{manager: tm}
		context.spinner = tm.sm.AddSpinner(message)
		context.Complete()
	}
}

// RunWithTrigger runs a task only if enabled.
func (tm *Manager) RunWithTrigger(enable bool, title string, callback func(task *Task)) {
	if enable {
		tm.Run(title, callback)
	}
}

// Run runs a synchronous task.
func (tm *Manager) Run(title string, callback func(task *Task)) {
	context := &Task{manager: tm, title: title}
	if tm.isOut && !tm.noProgress {
		context.spinner = tm.sm.AddSpinner(title)
	}

	callback(context)
}

// AsyncRun runs an asynchronous task.
func (tm *Manager) AsyncRun(title string, callback func(task *Task)) {
	context := &Task{manager: tm, title: title}
	if tm.isOut && !tm.noProgress {
		context.spinner = tm.sm.AddSpinner(title)
	}

	go callback(context)
}

// Complete marks the task as complete.
func (t *Task) Complete() {
	if t.manager.noProgress {
		return
	}

	if t.spinner == nil {
		return
	}

	t.spinner.Complete()
}

// Updatef updates the spinner message with format.
func (t *Task) Updatef(format string, a ...any) {
	if t.spinner == nil || t.manager.noProgress {
		return
	}

	t.spinner.UpdateMessagef(format, a...)
}

// Update updates the spinner message.
func (t *Task) Update(format string) {
	if t.spinner == nil || t.manager.noProgress {
		return
	}

	t.spinner.UpdateMessage(format)
}

// Println prints a message.
func (t *Task) Println(message string) {
	if t.manager.noProgress {
		_, _ = fmt.Fprintln(os.Stdout, message)

		return
	}

	if t.spinner == nil {
		return
	}

	t.spinner.UpdateMessage(message)
}

// Printf prints a formatted message.
func (t *Task) Printf(format string, args ...any) {
	if t.manager.noProgress {
		_, _ = fmt.Fprintf(os.Stdout, format+"\n", args...)

		return
	}

	if t.spinner == nil {
		return
	}

	t.spinner.UpdateMessagef(format, args...)
}

// CheckError ends the process with exit code 1 when err is not nil, or with 130 and an interrupted message when err
// wraps [context.Canceled], which the CLI's interrupt handling causes. The progress spinner shows the error when
// there is one. Otherwise the error goes to stderr, which keeps JSON and unix output on stdout intact.
//
// Parameters:
//   - err: the task's error, or nil to continue.
func (t *Task) CheckError(err error) {
	if err == nil {
		return
	}

	message := fmt.Sprintf("Fatal: %s, err: %v", strings.ToLower(t.title), err)
	code := exitFailure

	// The CLI cancels its context only on an interrupt, which is not a failure of the task.
	if errors.Is(err, context.Canceled) {
		message = "Interrupted"
		code = exitInterrupted
	}

	if t.spinner != nil {
		t.spinner.UpdateMessage(message)
		t.spinner.Error()
		t.manager.Stop()
	} else {
		_, _ = fmt.Fprintln(t.manager.stderr, message)
	}

	t.manager.exit(code)
}
