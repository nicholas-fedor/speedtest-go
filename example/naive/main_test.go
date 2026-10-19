// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test_checkError checks that checkError panics only for an error that is not nil.
func Test_checkError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantPanic bool
	}{
		{
			name:      "no error",
			err:       nil,
			wantPanic: false,
		},
		{
			name:      "error triggers panic",
			err:       errors.New("some error"),
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.wantPanic {
				assert.Panics(t, func() { checkError(tt.err) })
			} else {
				assert.NotPanics(t, func() { checkError(tt.err) })
			}
		})
	}
}
