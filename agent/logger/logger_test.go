package logger

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- handler output format ---

func TestHandler_InfoMessage(t *testing.T) {
	var buf bytes.Buffer
	h := newHandler(&buf, slog.LevelInfo)
	l := slog.New(h)

	l.Info("hello world")

	out := buf.String()
	if !strings.Contains(out, "INFO") {
		t.Errorf("expected INFO in output, got: %q", out)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected message in output, got: %q", out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("expected trailing newline, got: %q", out)
	}
}

func TestHandler_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	h := newHandler(&buf, slog.LevelWarn)
	l := slog.New(h)

	l.Debug("debug msg")
	l.Info("info msg")

	if buf.Len() != 0 {
		t.Errorf("expected no output for Debug/Info at WARN level, got: %q", buf.String())
	}

	l.Warn("warn msg")
	if !strings.Contains(buf.String(), "warn msg") {
		t.Errorf("expected warn msg in output, got: %q", buf.String())
	}
}

func TestHandler_AllLevels(t *testing.T) {
	levels := []struct {
		level slog.Level
		label string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
	}

	for _, tc := range levels {
		var buf bytes.Buffer
		h := newHandler(&buf, slog.LevelDebug)
		l := slog.New(h)
		l.Log(context.TODO(), tc.level, "msg")
		if !strings.Contains(buf.String(), tc.label) {
			t.Errorf("level %v: expected %q in output, got: %q", tc.level, tc.label, buf.String())
		}
	}
}

func TestHandler_Attrs(t *testing.T) {
	var buf bytes.Buffer
	h := newHandler(&buf, slog.LevelInfo)
	l := slog.New(h)

	l.Info("msg", "key", "value", "count", 42)

	out := buf.String()
	if !strings.Contains(out, "key=value") {
		t.Errorf("expected key=value in output, got: %q", out)
	}
	if !strings.Contains(out, "count=42") {
		t.Errorf("expected count=42 in output, got: %q", out)
	}
}

func TestHandler_AttrWithSpacesQuoted(t *testing.T) {
	var buf bytes.Buffer
	h := newHandler(&buf, slog.LevelInfo)
	l := slog.New(h)

	l.Info("msg", "path", "/a path/with spaces")

	out := buf.String()
	if !strings.Contains(out, `path="/a path/with spaces"`) {
		t.Errorf("expected quoted value with spaces, got: %q", out)
	}
}

func TestHandler_AttrWithQuoteEscaped(t *testing.T) {
	var buf bytes.Buffer
	h := newHandler(&buf, slog.LevelInfo)
	l := slog.New(h)

	l.Info("msg", "msg", `say "hi"`)

	out := buf.String()
	if !strings.Contains(out, `msg="say \"hi\""`) {
		t.Errorf("expected escaped quote in output, got: %q", out)
	}
}

// --- WithAttrs ---

func TestHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := newHandler(&buf, slog.LevelInfo)
	l := slog.New(h).With("component", "test")

	l.Info("started")

	out := buf.String()
	if !strings.Contains(out, "component=test") {
		t.Errorf("expected WithAttrs key in output, got: %q", out)
	}
}

// --- WithGroup (no-op) ---

func TestHandler_WithGroup_NoOp(t *testing.T) {
	var buf bytes.Buffer
	h := newHandler(&buf, slog.LevelInfo)
	l := slog.New(h).WithGroup("mygroup")

	l.Info("msg", "key", "val")

	out := buf.String()
	// Group prefix must NOT appear
	if strings.Contains(out, "mygroup") {
		t.Errorf("WithGroup should be a no-op, but group name appeared in output: %q", out)
	}
	// Attrs still present without prefix
	if !strings.Contains(out, "key=val") {
		t.Errorf("expected key=val in output even with group, got: %q", out)
	}
}

// --- parseLevel ---

func TestParseLevel_DefaultInfo(t *testing.T) {
	os.Unsetenv("WATCHFLARE_DEBUG")
	if parseLevel("") != slog.LevelInfo {
		t.Errorf("expected LevelInfo when WATCHFLARE_DEBUG unset and no cfgLevel")
	}
}

func TestParseLevel_DebugWhenEnvSet(t *testing.T) {
	os.Setenv("WATCHFLARE_DEBUG", "1")
	defer os.Unsetenv("WATCHFLARE_DEBUG")
	if parseLevel("") != slog.LevelDebug {
		t.Errorf("expected LevelDebug when WATCHFLARE_DEBUG set")
	}
}

func TestParseLevel_CfgLevelDebug(t *testing.T) {
	os.Unsetenv("WATCHFLARE_DEBUG")
	if parseLevel("debug") != slog.LevelDebug {
		t.Errorf("expected LevelDebug for cfgLevel=debug")
	}
}

func TestParseLevel_CfgLevelWarn(t *testing.T) {
	os.Unsetenv("WATCHFLARE_DEBUG")
	if parseLevel("warn") != slog.LevelWarn {
		t.Errorf("expected LevelWarn for cfgLevel=warn")
	}
}

// --- InitWithFile ---

func TestInitWithFile_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	if err := InitWithFile(path, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	slog.Info("file log test")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), "file log test") {
		t.Errorf("expected log message in file, got: %q", string(data))
	}

	// Restore stdout logger so other tests aren't affected
	Init()
}

func TestInitWithFile_InvalidPath_ReturnsError(t *testing.T) {
	err := InitWithFile("/nonexistent/path/test.log", "")
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}
