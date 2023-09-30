package logutil

import (
	"context"
	"log/slog"

	"github.com/cszatmary/goutils/errors"
)

// MultiHandler is a Handler that writes Records to multiple Handlers.
// It is useful for writing logs to multiple places, such as a file and stdout.
//
// Each handler can be configured independently to have different behaviour.
// For example, you could have one handler that writes text logs to stdout at info level or higher,
// while another writes JSON logs to a file at debug level or higher. This would allow for
// a simpler more human-friendly output on stdout, while still having the full logs available
// in a file for debugging.
type MultiHandler struct {
	handlers []slog.Handler
	opts     MultiHandlerOptions
}

type MultiHandlerOptions struct {
	// Level reports the minimum record level that will be logged.
	// If nil, the handler assumes slog.LevelDebug in order to allow
	// all handlers to receive the record and decide whether to handle it.
	// This should only be set if you know a certain level will never be used
	// by any handler and want to skip processing of that level.
	Level slog.Leveler
}

// NewMultiHandler creates a new MultiHandler that writes to the given handlers,
// using the given options. If opts is nil, the default options are used.
func NewMultiHandler(handlers []slog.Handler, opts *MultiHandlerOptions) *MultiHandler {
	if opts == nil {
		opts = &MultiHandlerOptions{}
	}
	return &MultiHandler{handlers: handlers, opts: *opts}
}

func (h *MultiHandler) Enabled(_ context.Context, level slog.Level) bool {
	// If no level is set, then the handler is always enabled so that each
	// individual handler can process the record.
	if h.opts.Level == nil {
		return true
	}
	return level >= h.opts.Level.Level()
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, h := range h.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: handlers, opts: h.opts}
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, h := range h.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: handlers, opts: h.opts}
}

// Handle calls Handle on each handler.
func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs errors.List
	for _, h := range h.handlers {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		if err := h.Handle(ctx, r); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}
