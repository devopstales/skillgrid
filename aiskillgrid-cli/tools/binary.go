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
)

// Downloader fetches data from a URL. Injectable for tests.
type Downloader func(url string) ([]byte, error)

// HTTPGet is the default Downloader using http.Get.
var HTTPGet Downloader = func(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	return io.ReadAll(resp.Body)
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

// EnsureReleaseBinary downloads a release asset if not already present and executable.
// Skips download if destName already exists in destDir and is executable (Unix: any
// execute bit set; Windows: any regular file).
func EnsureReleaseBinary(destDir, destName, assetURL string, get Downloader) error {
	if get == nil {
		get = HTTPGet
	}
	dest := filepath.Join(destDir, destName)

	if st, err := os.Stat(dest); err == nil && st.Mode().IsRegular() {
		if runtime.GOOS == "windows" || st.Mode()&0o111 != 0 {
			return nil
		}
	}

	data, err := get(assetURL)
	if err != nil {
		return err
	}

	return EnsureFileExecutable(dest, data)
}

// EngramAssetURL constructs a GitHub release URL for engram binary.
// In production, this may be resolved via GitHub API with injectable fetchJSON.
func EngramAssetURL(goos, goarch string) (string, error) {
	// For now, return a pattern that can be used in tests or with API resolution
	return fmt.Sprintf("https://github.com/engram-sh/engram/releases/latest/download/engram-%s-%s", goos, goarch), nil
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
