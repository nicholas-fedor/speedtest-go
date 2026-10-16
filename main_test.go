package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
)

// TestExitCode checks that interrupted runs exit with 130, invalid settings with 2, even when wrapped or joined, and
// other failures with 1.
func TestExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		name string
		want int
	}{
		{name: "success", err: nil, want: 0},
		{name: "failure", err: errors.New("network down"), want: 1},
		{
			name: "invalid setting",
			err:  fmt.Errorf("failed to load config: %w", config.ErrInvalidSetting),
			want: 2,
		},
		{
			name: "interrupted",
			err:  fmt.Errorf("failed to fetch servers: %w", context.Canceled),
			want: 130,
		},
		{
			name: "joined invalid setting",
			err: errors.Join(
				errors.New("other"),
				fmt.Errorf("%w --unit", config.ErrInvalidSetting),
			),
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, exitCode(tt.err))
		})
	}
}
