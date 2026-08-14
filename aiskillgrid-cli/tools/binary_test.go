package tools

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureReleaseBinaryWritesAndSkips(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	get := func(url string) ([]byte, error) {
		calls++
		return []byte("#!/bin/sh\necho ok\n"), nil
	}
	dest := filepath.Join(dir, "engram")
	if err := EnsureReleaseBinary(dir, "engram", "https://example.invalid/engram", get); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	st, err := os.Stat(dest)
	if err != nil || st.Mode()&0o111 == 0 {
		t.Fatalf("not executable: %v %#o", err, st.Mode())
	}
	if err := EnsureReleaseBinary(dir, "engram", "https://example.invalid/engram", get); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected skip, calls=%d", calls)
	}
}

func TestExtractBinaryFromTarGz(t *testing.T) {
	// Create in-memory tar.gz
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("#!/bin/sh\necho extracted\n")
	hdr := &tar.Header{
		Name: "engram",
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gw.Close()

	extracted, err := ExtractBinaryFromArchive(buf.Bytes(), "engram")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(extracted, content) {
		t.Fatalf("content mismatch")
	}
}

func TestExtractBinaryFromZip(t *testing.T) {
	// Create in-memory zip
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	content := []byte("@echo off\r\necho extracted\r\n")
	w, err := zw.Create("engram.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	zw.Close()

	extracted, err := ExtractBinaryFromArchive(buf.Bytes(), "engram.exe")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(extracted, content) {
		t.Fatalf("content mismatch")
	}
}

func TestExtractBinaryFromArchiveNotFound(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	tw.Close()
	gw.Close()

	_, err := ExtractBinaryFromArchive(buf.Bytes(), "missing")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
