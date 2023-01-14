package status

import (
	"os"
	"path/filepath"
	"time"
)

// Timestamp checks if any source change compared with the generated files,
// using file modifications timestamps.
type Timestamp struct {
	TempDir   string
	Task      string
	Dir       string
	Sources   []string
	Generates []string
	Dry       bool
}

// IsUpToDate implements the Checker interface
func (t *Timestamp) IsUpToDate() (bool, error) {
	if len(t.Sources) == 0 {
		return false, nil
	}

	sources, err := globs(t.Dir, t.Sources)
	if err != nil {
		return false, nil
	}
	generates, err := globs(t.Dir, t.Generates)
	if err != nil {
		return false, nil
	}

	timestampFile := t.timestampFilePath()

	// if the file exists, add the file path to the generates
	// if the generate file is old, the task will be executed
	_, err = os.Stat(timestampFile)
	if err == nil {
		generates = append(generates, timestampFile)
	} else {
		// create the timestamp file for the next execution when the file does not exist
		if !t.Dry {
			_ = os.MkdirAll(filepath.Dir(timestampFile), 0o755)
			_, _ = os.Create(timestampFile)
		}
	}

	taskTime := time.Now()

	// compare the time of the generates and sources. If the generates are old, the task will be executed

	// get the max time of the generates
	generateMaxTime, err := getMaxTime(generates...)
	if err != nil || generateMaxTime.IsZero() {
		return false, nil
	}

	// check if any of the source files is newer than the max time of the generates
	shouldUpdate, err := anyFileNewerThan(sources, generateMaxTime)
	if err != nil {
		return false, nil
	}

	// modify the metadata of the file to the the current time
	if !t.Dry {
		_ = os.Chtimes(timestampFile, taskTime, taskTime)
	}

	return !shouldUpdate, nil
}

func (t *Timestamp) Kind() string {
	return "timestamp"
}

// Value implements the Checker Interface
func (t *Timestamp) Value() (interface{}, error) {
	sources, err := globs(t.Dir, t.Sources)
	if err != nil {
		return time.Now(), err
	}

	sourcesMaxTime, err := getMaxTime(sources...)
	if err != nil {
		return time.Now(), err
	}

	if sourcesMaxTime.IsZero() {
		return time.Unix(0, 0), nil
	}

	return sourcesMaxTime, nil
}

func getMaxTime(files ...string) (time.Time, error) {
	var t time.Time
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return time.Time{}, err
		}
		t = maxTime(t, info.ModTime())
	}
	return t, nil
}

// if the modification time of any of the files is newer than the the given time, returns true
// This function is lazy, as it stops when it finds a file newer than the given time
func anyFileNewerThan(files []string, givenTime time.Time) (bool, error) {
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return false, err
		}
		if info.ModTime().After(givenTime) {
			return true, nil
		}
	}
	return false, nil
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// OnError implements the Checker interface
func (*Timestamp) OnError() error {
	return nil
}

func (t *Timestamp) timestampFilePath() string {
	return filepath.Join(t.TempDir, "timestamp", NormalizeFilename(t.Task))
}
