package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	t.Parallel()

	type args struct {
		jsonOutput bool
		unixOutput bool
	}

	tests := []struct {
		name string
		args args
		want *Manager
	}{
		{
			name: "json output true, unix false",
			args: args{jsonOutput: true, unixOutput: false},
			want: &Manager{isOut: false, noProgress: false},
		},
		{
			name: "json output false, unix false",
			args: args{jsonOutput: false, unixOutput: false},
			want: &Manager{isOut: true, noProgress: false},
		},
		{
			name: "json output true, unix true",
			args: args{jsonOutput: true, unixOutput: true},
			want: &Manager{isOut: true, noProgress: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewManager(tt.args.jsonOutput, tt.args.unixOutput)
			assert.NotNil(t, got)
			assert.Equal(t, tt.want.isOut, got.isOut)
			assert.Equal(t, tt.want.noProgress, got.noProgress)
		})
	}
}

func TestManager_Reset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tm   *Manager
	}{
		{
			name: "reset task manager",
			tm:   NewManager(false, false),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.tm.Reset() })
		})
	}
}

func TestManager_Stop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tm   *Manager
	}{
		{
			name: "stop task manager",
			tm:   NewManager(false, false),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.tm.Stop() })
		})
	}
}

func TestManager_Println(t *testing.T) {
	type args struct {
		message string
	}

	tests := []struct {
		name string
		tm   *Manager
		args args
	}{
		{
			name: "println message",
			tm:   NewManager(false, false),
			args: args{message: "test message"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.tm.Println(tt.args.message) })
		})
	}
}

func TestManager_RunWithTrigger(t *testing.T) {
	type args struct {
		enable   bool
		title    string
		callback func(task *Task)
	}

	tests := []struct {
		name string
		tm   *Manager
		args args
	}{
		{
			name: "run with trigger enabled",
			tm:   NewManager(false, false),
			args: args{
				enable:   true,
				title:    "test task",
				callback: func(_ *Task) {},
			},
		},
		{
			name: "run with trigger disabled",
			tm:   NewManager(false, false),
			args: args{
				enable:   false,
				title:    "test task",
				callback: func(_ *Task) {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(
				t,
				func() { tt.tm.RunWithTrigger(tt.args.enable, tt.args.title, tt.args.callback) },
			)
		})
	}
}

func TestManager_Run(t *testing.T) {
	type args struct {
		title    string
		callback func(task *Task)
	}

	tests := []struct {
		name string
		tm   *Manager
		args args
	}{
		{
			name: "run task",
			tm:   NewManager(false, false),
			args: args{
				title:    "test task",
				callback: func(_ *Task) {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.tm.Run(tt.args.title, tt.args.callback) })
		})
	}
}

func TestManager_AsyncRun(t *testing.T) {
	type args struct {
		title    string
		callback func(task *Task)
	}

	tests := []struct {
		name string
		tm   *Manager
		args args
	}{
		{
			name: "async run task",
			tm:   NewManager(false, false),
			args: args{
				title:    "test task",
				callback: func(_ *Task) {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.tm.AsyncRun(tt.args.title, tt.args.callback) })
		})
	}
}

func TestTask_Complete(t *testing.T) {
	tests := []struct {
		name string
		tr   *Task
	}{
		{
			name: "complete task",
			tr:   &Task{manager: NewManager(false, false)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.tr.Complete() })
		})
	}
}

func TestTask_Updatef(t *testing.T) {
	type args struct {
		format string
		a      []any
	}

	tests := []struct {
		name string
		tr   *Task
		args args
	}{
		{
			name: "updatef task",
			tr:   &Task{manager: NewManager(false, false)},
			args: args{
				format: "test %s",
				a:      []any{"format"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.tr.Updatef(tt.args.format, tt.args.a...) })
		})
	}
}

func TestTask_Update(t *testing.T) {
	type args struct {
		format string
	}

	tests := []struct {
		name string
		tr   *Task
		args args
	}{
		{
			name: "update task",
			tr:   &Task{manager: NewManager(false, false)},
			args: args{format: "test message"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.tr.Update(tt.args.format) })
		})
	}
}

func TestTask_Println(t *testing.T) {
	type args struct {
		message string
	}

	tests := []struct {
		name string
		tr   *Task
		args args
	}{
		{
			name: "println task",
			tr:   &Task{manager: NewManager(false, false)},
			args: args{message: "test message"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.tr.Println(tt.args.message) })
		})
	}
}

func TestTask_Printf(t *testing.T) {
	type args struct {
		format string
		a      []any
	}

	tests := []struct {
		name string
		tr   *Task
		args args
	}{
		{
			name: "printf task",
			tr:   &Task{manager: NewManager(false, false)},
			args: args{
				format: "test %s",
				a:      []any{"format"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.tr.Printf(tt.args.format, tt.args.a...) })
		})
	}
}

func TestTask_CheckError(t *testing.T) {
	type args struct {
		err error
	}

	tests := []struct {
		name string
		tr   *Task
		args args
	}{
		{
			name: "check error nil",
			tr:   &Task{manager: NewManager(false, false), title: "test"},
			args: args{err: nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.tr.CheckError(tt.args.err) })
		})
	}
}
