package tools

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	downloadTimeout = 60 * time.Second
	maxDownloadSize = 200 << 20
)

// Downloader fetches data from a URL. Injectable for tests.
type Downloader func(url string) ([]byte, error)

// HTTPGet is the default Downloader. It bounds both wall time and response size
// so a hung or hostile endpoint cannot stall or exhaust memory during install.
var HTTPGet Downloader = func(url string) ([]byte, error) {
	client := &http.Client{Timeout: downloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDownloadSize {
		return nil, fmt.Errorf("response exceeds %d byte limit: %s", maxDownloadSize, url)
	}
	return data, nil
}

// EnsureFileExecutable writes data to path with executable permissions.
func EnsureFileExecutable(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// BinaryInstalled reports whether path is a regular file that can be executed
// (Unix: any execute bit set; Windows: any regular file).
func BinaryInstalled(path string) bool {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return false
	}
	return runtime.GOOS == "windows" || st.Mode()&0o111 != 0
}

// EnsureReleaseBinary downloads a release asset into destDir/destName unless an
// executable copy is already there. Release assets are commonly tar.gz or zip
// archives, so the payload is unpacked when destName is found inside it and
// written verbatim otherwise.
func EnsureReleaseBinary(destDir, destName, assetURL string, get Downloader) error {
	if get == nil {
		get = HTTPGet
	}
	dest := filepath.Join(destDir, destName)
	if BinaryInstalled(dest) {
		return nil
	}

	data, err := get(assetURL)
	if err != nil {
		return err
	}
	if extracted, err := ExtractBinaryFromArchive(data, destName); err == nil {
		data = extracted
	}

	return EnsureFileExecutable(dest, data)
}

// ExtractBinaryFromArchive extracts a named file from tar.gz or zip archive.
// Returns the file content or error if not found.
func ExtractBinaryFromArchive(data []byte, wantName string) ([]byte, error) {
	// Try tar.gz first
	if content, err := extractFromTarGz(data, wantName); err == nil {
		return content, nil
	}

	// Try zip
	if content, err := extractFromZip(data, wantName); err == nil {
		return content, nil
	}

	return nil, fmt.Errorf("file %q not found in archive", wantName)
}

func extractFromTarGz(data []byte, wantName string) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == wantName || filepath.Base(hdr.Name) == wantName {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("not found in tar.gz")
}

func extractFromZip(data []byte, wantName string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	for _, f := range r.File {
		if f.Name == wantName || filepath.Base(f.Name) == wantName {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("not found in zip")
}
