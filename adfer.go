package adfer

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"
)

// CrashReport represents a single crash report
type CrashReport struct {
	Timestamp  time.Time         `json:"timestamp"`
	Error      string            `json:"error"`
	Stack      string            `json:"stack"`
	SystemInfo SystemInfo        `json:"system_info,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// SystemInfo represents system information
type SystemInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	GoVersion    string `json:"go_version"`
}

// ErrorHandler is a function type for custom error handling
type ErrorHandler func(error, []byte)

// Options struct holds the configuration for panic handling
type Options struct {
	// ErrorHandler is a custom error handling function
	ErrorHandler ErrorHandler
	// DumpToFile enables dumping errors to a file
	DumpToFile bool
	// FilePath is the path to the file to dump errors to
	FilePath string
	// ExitOnPanic enables exiting the program after handling a panic
	ExitOnPanic bool
	// IncludeSystemInfo enables including system information in crash reports
	IncludeSystemInfo bool
	// Metadata is custom metadata to include in crash reports
	Metadata map[string]string
	// WipeFile enables wiping the crash file on initialization
	WipeFile bool
}

type PanicHandler struct {
	options  Options
	exitFunc func(int)
}

// defaultErrorHandler is the default error handling function
func defaultErrorHandler(err error, stack []byte) {
	fmt.Printf("Recovered from panic:\nError: %v\nStack Trace:\n%s\n", err, stack)
}

// New initializes a new PanicHandler with optional configurations
func New(options Options) *PanicHandler {
	if options.ErrorHandler == nil {
		options.ErrorHandler = defaultErrorHandler
	}
	ph := &PanicHandler{
		options:  options,
		exitFunc: os.Exit,
	}
	if ph.options.WipeFile && ph.options.DumpToFile {
		err := ph.WipeCrashFile()
		if err != nil {
			fmt.Printf("Error wiping crash file: %v\n", err)
		}
	}
	return ph
}

// Recover is the main function to recover from panics
func (ph *PanicHandler) Recover() {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}
		stack := debug.Stack()
		ph.options.ErrorHandler(err, stack)

		if ph.options.DumpToFile {
			report := CrashReport{
				Timestamp: time.Now(),
				Error:     err.Error(),
				Stack:     string(stack),
				Metadata:  ph.options.Metadata,
			}

			if ph.options.IncludeSystemInfo {
				report.SystemInfo = SystemInfo{
					OS:           runtime.GOOS,
					Architecture: runtime.GOARCH,
					GoVersion:    runtime.Version(),
				}
			}

			ph.appendCrashReport(report)
		}

		if ph.options.ExitOnPanic {
			ph.exitFunc(1)
		}
	}
}

func (ph *PanicHandler) appendCrashReport(report CrashReport) {
	var reports []CrashReport

	data, err := os.ReadFile(ph.options.FilePath)
	if err == nil {
		err := json.Unmarshal(data, &reports)
		if err != nil {
			fmt.Printf("Error unmarshalling crash reports: %v\n", err)
		}
	}

	reports = append(reports, report)

	data, _ = json.MarshalIndent(reports, "", "  ")
	err = os.WriteFile(ph.options.FilePath, data, 0644)
	if err != nil {
		fmt.Printf("Error writing crash report to file: %v\n", err)
	}
}

// SafeGo wraps a function to be executed in a goroutine with panic recovery
func (ph *PanicHandler) SafeGo(f func()) {
	go func() {
		defer ph.Recover()
		f()
	}()
}

// GetLastNCrashReports retrieves the last N crash reports from the log file
func (ph *PanicHandler) GetLastNCrashReports(n int) ([]CrashReport, error) {
	if ph.options.FilePath == "" {
		return nil, fmt.Errorf("no file path set for crash reports")
	}
	data, err := os.ReadFile(ph.options.FilePath)
	if err != nil {
		return nil, err
	}

	var reports []CrashReport
	err = json.Unmarshal(data, &reports)
	if err != nil {
		return nil, err
	}

	if len(reports) <= n {
		return reports, nil
	}
	return reports[len(reports)-n:], nil
}

// WipeCrashFile clears all crash reports from the log file
func (ph *PanicHandler) WipeCrashFile() error {
	if ph.options.FilePath == "" {
		return fmt.Errorf("no file path set for crash reports")
	}
	return os.WriteFile(ph.options.FilePath, []byte("[]"), 0644)
}
