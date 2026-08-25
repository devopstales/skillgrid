package logging

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	once   sync.Once
	logPath string
	mu     sync.Mutex
)

func ResetForTest() {
	once = sync.Once{}
}

func Init(baseDir string) error {
	var err error
	once.Do(func() {
		logPath = filepath.Join(baseDir, "logs", "install.log")
		if err != nil {
			return
		}
		if mkErr := os.MkdirAll(filepath.Dir(logPath), 0755); mkErr != nil {
			err = mkErr
			return
		}
		f, ferr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if ferr != nil {
			err = ferr
			return
		}
		f.Close()
	})
	return err
}

func Path() string {
	return logPath
}

func write(level, msg string) {
	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	line := time.Now().Format(time.RFC3339) + " [" + level + "] " + msg + "\n"
	f.WriteString(line)
}

func Info(msg string)  { write("INFO", msg) }
func Warn(msg string)  { write("WARN", msg) }
func Error(msg string) { write("ERROR", msg) }
