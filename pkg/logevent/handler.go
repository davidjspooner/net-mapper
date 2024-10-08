package logevent

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const EventAttrKey = "event"

var eventCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "logged_events",
	Help: "Count logged events",
}, []string{"level", "group", "event"})

type handler struct {
	opt   *slog.HandlerOptions
	attrs []slog.Attr
	group []string
}

func NewHandler(opt *slog.HandlerOptions) slog.Handler {
	return &handler{opt: opt}
}

var _ slog.Handler = &handler{}

// Enabled implements slog.Handler.
func (l *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return true //pass all logs to handler so events can be counted ( and then discarded if under log level )
}

// Handle implements slog.Handler.
func (l *handler) Handle(ctx context.Context, r slog.Record) error {

	attr := make(map[string]any)
	level := r.Level
	var event string

	attrFunc := func(a slog.Attr) bool {
		key := a.Key
		i := a.Value.Any()
		if i == nil {
			return true
		}
		if key == EventAttrKey {
			event = a.Value.String()
			return true
		}
		attr[key] = a.Value.String()
		return true
	}

	for _, a := range l.attrs {
		attrFunc(a)
	}
	r.Attrs(attrFunc)

	group := "/" + strings.Join(l.group, "/")
	if len(l.group) > 0 {
		group += "/"
	}

	if event != "" {
		eventCounter.WithLabelValues(level.String(), group, event).Inc()
		group += event
		if level <= l.opt.Level.Level() {
			return nil
		}
	}

	line := []any{r.Time.Format(time.RFC1123Z), level.String(), group, r.Message, attr}

	e := json.NewEncoder(os.Stdout)
	e.SetEscapeHTML(false)
	err := e.Encode(line)
	return err
}

// WithAttrs implements slog.Handler.
func (l *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	copy := &handler{opt: l.opt}
	copy.attrs = append(copy.attrs, l.attrs...)
	copy.attrs = append(copy.attrs, attrs...)
	copy.group = append(copy.group, l.group...)
	return copy
}

// WithGroup implements slog.Handler.
func (l *handler) WithGroup(name string) slog.Handler {
	copy := &handler{opt: l.opt}
	copy.attrs = append(copy.attrs, l.attrs...)
	copy.group = append(copy.group, l.group...)
	copy.group = append(copy.group, name)
	return copy
}

var ctxKey = &handler{}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey, logger)
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	logger, _ := ctx.Value(ctxKey).(*slog.Logger)
	return logger
}
