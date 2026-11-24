package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogWritesJSONLines(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(dir, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	if err := logger.Log(Entry{Tool: "bills_list", Result: "ok", Subject: "owner", DurationMS: 12, RequestID: "abc"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{`"tool":"bills_list"`, `"result":"ok"`, `"subject":"owner"`, `"request_id":"abc"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("falta %s en %s", want, text)
		}
	}
	info, err := os.Stat(filepath.Join(dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permisos = %o", info.Mode().Perm())
	}
}

func TestRotation(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(dir, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	logger.maxBytes = 80
	for i := 0; i < 5; i++ {
		if err := logger.Log(Entry{Tool: "echo", Result: "ok"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "audit.1.jsonl")); err != nil {
		t.Fatalf("no rotó: %v", err)
	}
}

func TestRejectsEmptyDir(t *testing.T) {
	if _, err := New("", 5); err == nil {
		t.Fatal("quiero error con dir vacío")
	}
}
