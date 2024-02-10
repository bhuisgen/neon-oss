package log

import (
	"bytes"
	"fmt"
	"log/slog"
	"testing"
	"testing/slogtest"
)

func TestLogHandler_Default(t *testing.T) {
	var buf bytes.Buffer
	if err := slogtest.TestHandler(NewHandler(&buf, "test", nil), func() []map[string]any {
		return parseLogEntries(t, buf.Bytes())
	}); err != nil {
		t.Error(err)
	}
}

func TestLogHandler_CustomOptions(t *testing.T) {
	var buf bytes.Buffer
	if err := slogtest.TestHandler(NewHandler(&buf, "test", &HandlerOptions{
		Level: slog.LevelInfo,
	}), func() []map[string]any {
		return parseLogEntries(t, buf.Bytes())
	}); err != nil {
		t.Error(err)
	}
}

func parseLogEntries(t *testing.T, data []byte) []map[string]any {
	ms := []map[string]any{}
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		m := map[string]any{}
		for _, field := range splitFields(line) {
			key, value, found := bytes.Cut(field, []byte{'='})
			if !found || len(key) == 0 || len(value) == 0 {
				t.Fatal(fmt.Errorf("failed to parse field '%s' for line '%s'", string(field), string(line)))
			}
			keyItems := bytes.Split(key, []byte{'.'})

			switch i := len(keyItems); {
			case i == 2:
				group := string(keyItems[0])
				if m[group] == nil {
					m[group] = map[string]any{}
				}
				m[group].(map[string]any)[string(keyItems[len(keyItems)-1])] = string(bytes.Trim(value, "\""))
			case i > 2:
				groups := keyItems[:len(keyItems)-1]
				var mg map[string]any = m
				for _, g := range groups {
					group := string(g)
					if mg[group] == nil {
						mg[group] = map[string]any{}
					}
					mg = mg[group].(map[string]any)
				}
				mg[string(keyItems[len(keyItems)-1])] = string(bytes.Trim(value, "\""))
			default:
				m[string(key)] = string(bytes.Trim(value, "\""))
			}
		}
		ms = append(ms, m)
	}
	return ms
}

func splitFields(b []byte) [][]byte {
	var quoted bool
	return bytes.FieldsFunc(b, func(r1 rune) bool {
		if r1 == '"' {
			quoted = !quoted
		}
		return !quoted && r1 == ' '
	})
}
