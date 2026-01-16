// Package task provides a task manager for CLI output with progress spinners.
package task

import (
	"fmt"
	"os"
	"strings"

	"github.com/chelnak/ysmrr"
)

// Manager manages tasks with spinners for CLI output.
type Manager struct {
	sm         ysmrr.SpinnerManager
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

	tm := &Manager{sm: ysmrr.NewSpinnerManager(), isOut: isOut, noProgress: unixOutput}
	if isOut && !unixOutput {
		tm.sm.Start()
	}

	return tm
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

// CheckError checks for error and exits if present.
func (t *Task) CheckError(err error) {
	if err != nil {
		if t.spinner != nil {
			t.Printf("Fatal: %s, err: %v", strings.ToLower(t.title), err)
			t.spinner.Error()
			t.manager.Stop()
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "Fatal: %s, err: %v", strings.ToLower(t.title), err)
		}

		os.Exit(0)
	}
}
