package slg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"sync"
)

const (
	reset = "\033[0m"

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97
	timeFormat   = "[2006-01-02 15:04:05 IST]"
)

func colorize(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

type simpleLogHandler struct {
	opts Options
	// TODO: state for WithGroup and WithAttrs
	mu       *sync.Mutex
	out      io.Writer
	filename string
}
type Options struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler
	Ansi  bool
}

func CreateHandler(out io.Writer, filename string, opts *Options) *simpleLogHandler {
	h := &simpleLogHandler{out: out, mu: &sync.Mutex{}, filename: filename}
	if opts != nil {
		h.opts = *opts
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}
	return h
}

func (h *simpleLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// !-enabled
func (h *simpleLogHandler) WithGroup(name string) slog.Handler {
	// TODO: implement.
	return h
}

func (h *simpleLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// TODO: implement.
	return h
}

func (h *simpleLogHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)
	//get sting values
	timestr := r.Time.Format(timeFormat)
	msg := r.Message
	level := r.Level.String()
	fsline := "0"

	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		fsline = fmt.Sprint(f.Line)
	}

	if h.opts.Ansi {
		timestr = colorize(magenta, timestr)

		switch r.Level {
		case slog.LevelDebug:
			level = colorize(darkGray, level)
		case slog.LevelInfo:
			level = colorize(lightGreen, level)
		case slog.LevelWarn:
			level = colorize(lightYellow, level)
		case slog.LevelError:
			level = colorize(lightRed, level)
		}
		fsline = colorize(yellow, string(fsline))
		msg = colorize(lightCyan, msg)
	}

	if !r.Time.IsZero() {
		buf = fmt.Appendf(buf, "%s", timestr)
	}

	buf = fmt.Appendf(buf, " %-14s", level)

	buf = fmt.Appendf(buf, fmt.Sprintf(" %s:%s", h.filename, fsline))
	buf = fmt.Appendf(buf, " %s", msg)
	r.Attrs(
		func(a slog.Attr) bool {

			buf = fmt.Appendf(buf, " %s = %s", a.Key, a.Value)
			return true
		})

	buf = append(buf, "\n"...)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err

}
