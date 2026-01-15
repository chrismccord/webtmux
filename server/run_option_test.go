package server

import (
	"context"
	"testing"
)

func TestRunOptionsDefault(t *testing.T) {
	opts := &RunOptions{}

	if opts.gracefullCtx != nil {
		t.Error("gracefullCtx should be nil by default")
	}
}

func TestWithGracefullContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := &RunOptions{}
	option := WithGracefullContext(ctx)

	// Apply the option
	option(opts)

	if opts.gracefullCtx != ctx {
		t.Error("WithGracefullContext should set gracefullCtx")
	}
}

func TestWithGracefullContextNil(t *testing.T) {
	opts := &RunOptions{}
	option := WithGracefullContext(nil)

	// Apply the option - should not panic
	option(opts)

	if opts.gracefullCtx != nil {
		t.Error("gracefullCtx should be nil when passed nil")
	}
}

func TestMultipleRunOptions(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	opts := &RunOptions{}

	// Apply multiple options - last one wins
	WithGracefullContext(ctx1)(opts)
	WithGracefullContext(ctx2)(opts)

	if opts.gracefullCtx != ctx2 {
		t.Error("Last applied context should be used")
	}
}
