package backend

import (
	"context"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/chat"
	"github.com/rk-senne/moonbase/internal/discovery"
)

// oneShotStream wraps a non-streaming Backend.Deploy call as a single-chunk
// stream: one chunk with the full output text (if any), then a terminal Done
// chunk. This provides a uniform channel-based interface for all backends.
func oneShotStream(ctx context.Context, be Backend, agent agents.Agent,
	pc *discovery.ProjectContext, task string) <-chan chat.StreamChunk {
	ch := make(chan chat.StreamChunk, 2)
	go func() {
		defer close(ch)
		// Early out if context is already cancelled
		if ctx.Err() != nil {
			ch <- chat.StreamChunk{Done: true, Err: ctx.Err()}
			return
		}
		out, err := be.Deploy(agent, pc, task)
		if ctx.Err() != nil {
			ch <- chat.StreamChunk{Done: true, Err: ctx.Err()}
			return
		}
		if err == nil && out != "" {
			ch <- chat.StreamChunk{Text: out}
		}
		ch <- chat.StreamChunk{Done: true, Err: err}
	}()
	return ch
}
