package config

import (
	"os"
	"testing"
)

func createTempFile(f string, p string) string {
	tmpFile, err := os.Create(os.TempDir() + "/" + "ssh_ms_test." + p)
	text := ""
	if err != nil {
		return ""
	}

	switch f {
	case "json":
		text += `{"test": 1}`
	case "yaml":
		text += "---\ntest: True\n..."
	default:
		text += "Test"
	}
	content := []byte(text)
	if _, err := tmpFile.Write(content); err != nil {
		return ""
	}
	defer tmpFile.Close()
	return tmpFile.Name()
}

func TestGetFileType(t *testing.T) {
	// Test basic functionality
	tf := createTempFile("yaml", "yaml")
	yf, err := os.Open(tf)
	if err != nil {
		t.Fatalf("Failed to open temporary file %v", yf)
	} else if _, err := GetFileType(yf); err != nil {
		t.Fatalf("Failed to GetFileType %v", err)
	}
	defer os.Remove(tf)

	// Test expected mimetype
	tf = createTempFile("json", "json")
	yf, err = os.Open(tf)
	if err != nil {
		t.Fatalf("Failed to open temporary file %v", yf)
	} else if ct, _ := GetFileType(yf); ct != FormatJSON {
		t.Fatalf("expected: '%v' got: '%v'", FormatJSON, ct)
	}
	defer os.Remove(tf)

	// Test missing files
	tf = createTempFile("json", "json")
	yf, err = os.Open(tf)
	yf.Close()
	os.Remove(tf)
	if err != nil {
		t.Fatalf("Failed to open temporary file %v", yf)
	} else if ct, _ := GetFileType(yf); ct != FormatUnknown {
		t.Fatalf("expected: '%v' got: '%v'", FormatUnknown, ct)
	}
}

func TestEnsureDirExists(t *testing.T) {
	ts := "/ensureDirExists"
	td := os.TempDir() + ts
	defer os.RemoveAll(td)

	if ok, err := ensureDirExists(td); !ok {
		t.Fatalf("expected: %v exists, got: %v", td, err)
	}

	if ok, err := ensureDirExists(ts); ok {
		t.Fatalf("expected: %v to not exist, got: %v", ts, err)
	}
}
