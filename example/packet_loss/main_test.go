package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_checkError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "no error",
			err:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { checkError(tt.err) })
		})
	}
}
