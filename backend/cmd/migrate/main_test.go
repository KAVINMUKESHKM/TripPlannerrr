package main

import (
	"context"
	"testing"
)

func TestRunRequiresUpSubcommand(t *testing.T) {
	for _, args := range [][]string{nil, {"down"}, {"up", "extra"}} {
		if err := run(context.Background(), args); err == nil {
			t.Errorf("run(%v) succeeded, want usage error", args)
		}
	}
}
