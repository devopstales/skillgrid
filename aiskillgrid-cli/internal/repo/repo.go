package repo

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// DefaultRepoURL is the fallback clone source (docs/00-aiskillgrid-cli.md).
const DefaultRepoURL = "https://github.com/devopstales/aiskillgrid.git"

// Sync copies a local repo checkout into baseDir/repos/aiskillgrid and
// refreshes baseDir/config.d from its config.d directory.
func Sync(src, baseDir string) error {
	src, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	if fi, err := os.Stat(src); err != nil || !fi.IsDir() {
		return fmt.Errorf("sync source not found: %s", src)
	}
	dst := filepath.Join(baseDir, "repos", "aiskillgrid")
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	if err := copyDir(src, dst); err != nil {
		return fmt.Errorf("copy repo: %w", err)
	}
	srcCfg := filepath.Join(src, "config.d")
	dstCfg := filepath.Join(baseDir, "config.d")
	if fi, err := os.Stat(srcCfg); err == nil && fi.IsDir() {
		if err := os.MkdirAll(dstCfg, 0755); err != nil {
			return err
		}
		if err := copyDir(srcCfg, dstCfg); err != nil {
			return fmt.Errorf("copy config.d: %w", err)
		}
	}
	return nil
}

// Clone git-clones the repo into baseDir/repos/aiskillgrid and copies its
// config.d into baseDir/config.d.
func Clone(baseDir, url string) error {
	dst := filepath.Join(baseDir, "repos", "aiskillgrid")
	if fi, err := os.Stat(dst); err == nil && fi.IsDir() {
		cmd := exec.Command("git", "-C", dst, "pull")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git pull: %w", err)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		cmd := exec.Command("git", "clone", url, dst)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git clone: %w", err)
		}
	}
	srcCfg := filepath.Join(dst, "config.d")
	dstCfg := filepath.Join(baseDir, "config.d")
	if fi, err := os.Stat(srcCfg); err == nil && fi.IsDir() {
		if err := os.MkdirAll(dstCfg, 0755); err != nil {
			return err
		}
		if err := copyDir(srcCfg, dstCfg); err != nil {
			return fmt.Errorf("copy config.d: %w", err)
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		fi, lerr := os.Lstat(p)
		if lerr == nil && fi.Mode()&os.ModeSymlink != 0 {
			lnk, err := os.Readlink(p)
			if err != nil {
				return err
			}
			os.Remove(target)
			return os.Symlink(lnk, target)
		}
		switch {
		case info.IsDir():
			return os.MkdirAll(target, info.Mode().Perm())
		default:
			return copyFile(p, target)
		}
	})
}

func copyFile(src, dst string) error {
	os.Remove(dst)
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(out, in)
	return err
}
