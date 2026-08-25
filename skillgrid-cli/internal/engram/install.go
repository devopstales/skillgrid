package engram

import (
	"skillgrid-cli/internal/logging"
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func DetectPlatform() (string, string, error) {
	goos := runtime.GOOS
	arch := runtime.GOARCH
	if goos == "darwin" && arch == "arm64" {
		return "darwin", "arm64", nil
	}
	if goos == "darwin" && arch == "amd64" {
		return "darwin", "amd64", nil
	}
	if goos == "linux" && arch == "arm64" {
		return "linux", "arm64", nil
	}
	if goos == "linux" && arch == "amd64" {
		return "linux", "amd64", nil
	}
	return "", "", fmt.Errorf("unsupported platform: %s/%s", goos, arch)
}

func InstallBinary(baseDir string) error {
	binDir := filepath.Join(baseDir, "bin")
	binaryPath := filepath.Join(binDir, "engram")
	if _, err := os.Stat(binaryPath); err == nil {
		return nil
	}

	goos, arch, err := DetectPlatform()
	if err != nil {
		return err
	}

	if err := logging.Init(baseDir); err != nil {
		return err
	}
	logging.Info("installing engram binary for " + goos + "/" + arch)

	version, err := fetchLatestVersion()
	if err != nil {
		logging.Error("fetch engram version failed: " + err.Error())
		return err
	}

	asset := "engram_" + version + "_" + goos + "_" + arch + ".tar.gz"
	url := "https://github.com/Gentleman-Programming/engram/releases/download/v" + version + "/" + asset
	tmpFile := filepath.Join(baseDir, "tmp", "engram.tar.gz")
	if err := downloadFile(tmpFile, url); err != nil {
		logging.Error("download engram failed: " + err.Error())
		return err
	}

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	if err := extractTarGz(tmpFile, binDir); err != nil {
		logging.Error("extract engram failed: " + err.Error())
		return err
	}

	if err := os.Chmod(binaryPath, 0755); err != nil {
		return err
	}

	logging.Info("engram binary installed to " + binaryPath)
	return nil
}

func fetchLatestVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/Gentleman-Programming/engram/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	tag := strings.TrimPrefix(strings.Split(string(body), `"tag_name":"`)[1], "v")
	tag = strings.Split(tag, `"`)[0]
	return tag, nil
}

func downloadFile(path, url string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTarGz(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.Base(hdr.Name))
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, tr)
		out.Close()
		if err != nil {
			return err
		}
		os.Chmod(target, hdr.FileInfo().Mode().Perm())
	}
	return nil
}
