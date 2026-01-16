package speedtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDebug(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "create new debug",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewDebug()
			assert.NotNil(t, got)
			assert.False(t, got.flag)
			assert.NotNil(t, got.dbg)
		})
	}
}

func TestDebug_Enable(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "enable debug",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := NewDebug()
			assert.False(t, d.flag)
			d.Enable()
			assert.True(t, d.flag)
		})
	}
}

func TestDebug_Println(t *testing.T) {
	tests := []struct {
		name string
		v    []any
	}{
		{
			name: "println with values",
			v:    []any{"test", 123},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := NewDebug()
			// Initially disabled, should not print
			d.Println(tt.v...)
			// After enable, should print
			d.Enable()
			d.Println(tt.v...)
		})
	}
}

func TestDebug_Printf(t *testing.T) {
	tests := []struct {
		name   string
		format string
		v      []any
	}{
		{
			name:   "printf with format",
			format: "test %s %d",
			v:      []any{"value", 456},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := NewDebug()
			// Initially disabled
			d.Printf(tt.format, tt.v...)
			// After enable
			d.Enable()
			d.Printf(tt.format, tt.v...)
		})
	}
}
