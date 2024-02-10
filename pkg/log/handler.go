package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// Handler implements the log handler.
type Handler struct {
	id   string
	opts *HandlerOptions
	w    io.Writer
	mu   *sync.Mutex
	goas []groupOrAttrs
}

// HandlerOptions implements the log handler options.
type HandlerOptions struct {
	Level        slog.Leveler
	AppendSource bool
}

// groupOrAttrs holds either the group or the list of attributes.
type groupOrAttrs struct {
	group string
	attrs []slog.Attr
}

const (
	// IDKey is the key used by the handler for its ID. The associated value is a
	// string.
	IDKey = "id"
)

// NewHandler creates a new handler.
func NewHandler(w io.Writer, id string, opts *HandlerOptions) *Handler {
	h := Handler{
		id: id,
		w:  w,
		mu: &sync.Mutex{},
	}
	if opts == nil {
		h.opts = &HandlerOptions{
			Level: ProgramLevel,
		}
	} else {
		h.opts = opts
	}
	return &h
}

// Enabled reports whether the handler handles records at the given level.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

// WithAttrs returns a new Handler whose attributes consist of both the
// receiver's attributes and the arguments.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

// Handle handles the Record.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)
	if !r.Time.IsZero() {
		buf = h.appendAttr(buf, "", slog.Time(slog.TimeKey, r.Time))
	}
	buf = h.appendAttr(buf, "", slog.Any(slog.LevelKey, r.Level))
	if h.id != "" {
		buf = h.appendAttr(buf, "", slog.String(IDKey, h.id))
	}
	if h.opts.AppendSource {
		if r.PC != 0 {
			fs := runtime.CallersFrames([]uintptr{r.PC})
			f, _ := fs.Next()
			buf = h.appendAttr(buf, "", slog.String(slog.SourceKey, fmt.Sprintf("%s:%d", f.File, f.Line)))
		}
	}
	buf = h.appendAttr(buf, "", slog.String(slog.MessageKey, r.Message))
	prefix := ""
	goas := h.goas
	if r.NumAttrs() == 0 {
		for len(goas) > 0 && goas[len(goas)-1].group != "" {
			goas = goas[:len(goas)-1]
		}
	}
	for _, goa := range goas {
		if goa.group != "" {
			prefix += goa.group + "."
		} else {
			for _, a := range goa.attrs {
				buf = h.appendAttr(buf, prefix, a)
			}
		}
	}
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, prefix, a)
		return true
	})
	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	if _, err := h.w.Write(buf); err != nil {
		return fmt.Errorf("write record: %w", err)
	}

	return nil
}

// withGroupOrAttrs creates a new handler with the given group or attributes.
func (h *Handler) withGroupOrAttrs(goa groupOrAttrs) *Handler {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

// appendAttr appends a single attribute.
func (h *Handler) appendAttr(buf []byte, prefix string, a slog.Attr) []byte {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf
	}
	if prefix != "" {
		a.Key = prefix + a.Key
	}
	switch a.Value.Kind() {
	case slog.KindString:
		if needsQuoting(a.Key) {
			buf = fmt.Appendf(buf, " %q=", a.Key)
		} else {
			buf = fmt.Appendf(buf, " %s=", a.Key)
		}
		if needsQuoting(a.Value.String()) {
			buf = fmt.Appendf(buf, "%q", a.Value)
		} else {
			buf = fmt.Appendf(buf, "%s", a.Value)
		}
	case slog.KindTime:
		buf = fmt.Appendf(buf, "%s=%s", a.Key, a.Value.Time().Format(time.RFC3339Nano))
	case slog.KindGroup:
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return buf
		}
		for _, ga := range attrs {
			if a.Key == "" {
				buf = h.appendAttr(buf, prefix, ga)
			} else {
				buf = h.appendAttr(buf, a.Key+".", ga)
			}
		}
	default:
		if needsQuoting(a.Key) {
			buf = fmt.Appendf(buf, " %q=", a.Key)
		} else {
			buf = fmt.Appendf(buf, " %s=", a.Key)
		}
		if needsQuoting(a.Value.String()) {
			buf = fmt.Appendf(buf, "%q", a.Value.String())
		} else {
			buf = fmt.Appendf(buf, "%s", a.Value.String())
		}
	}
	return buf
}

// needsQuoting checks if a string needs to be quoted.
func needsQuoting(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return true
		}
		i += size
	}
	return false
}

var _ (slog.Handler) = (*Handler)(nil)
