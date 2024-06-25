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

// PanicHandler struct holds the configuration for panic handling
type PanicHandler struct {
	errorHandler      ErrorHandler
	dumpToFile        bool
	filePath          string
	exitOnPanic       bool
	includeSystemInfo bool
	metadata          map[string]string
	wipeFile          bool
}

// Option is a function type for functional options
type Option func(*PanicHandler)

// defaultErrorHandler is the default error handling function
func defaultErrorHandler(err error, stack []byte) {
	fmt.Printf("Recovered from panic:\nError: %v\nStack Trace:\n%s\n", err, stack)
}

// New initializes a new PanicHandler with optional configurations
func New(options ...Option) *PanicHandler {
	ph := &PanicHandler{
		errorHandler:      defaultErrorHandler,
		dumpToFile:        false,
		filePath:          "panic.log",
		exitOnPanic:       false,
		includeSystemInfo: false,
		metadata:          make(map[string]string),
		wipeFile:          false,
	}
	for _, option := range options {
		option(ph)
	}
	if ph.wipeFile && ph.dumpToFile {
		err := ph.WipeCrashFile()
		if err != nil {
			fmt.Printf("Error wiping crash file: %v\n", err)
		}
	}
	return ph
}

// WithErrorHandler sets a custom error handler
func WithErrorHandler(handler ErrorHandler) Option {
	return func(ph *PanicHandler) {
		ph.errorHandler = handler
	}
}

// WithDumpToFile enables dumping errors to a file
func WithDumpToFile(filePath string) Option {
	return func(ph *PanicHandler) {
		ph.dumpToFile = true
		ph.filePath = filePath
	}
}

// WithExitOnPanic enables exiting the program after handling a panic
func WithExitOnPanic() Option {
	return func(ph *PanicHandler) {
		ph.exitOnPanic = true
	}
}

// WithSystemInfo enables including system information in crash reports
func WithSystemInfo() Option {
	return func(ph *PanicHandler) {
		ph.includeSystemInfo = true
	}
}

// WithMetadata adds custom metadata to crash reports
func WithMetadata(metadata map[string]string) Option {
	return func(ph *PanicHandler) {
		for k, v := range metadata {
			ph.metadata[k] = v
		}
	}
}

// WithWipeFile enables wiping the crash file on initialization
func WithWipeFile() Option {
	return func(ph *PanicHandler) {
		ph.wipeFile = true
	}
}

// Recover is the main function to recover from panics
func (ph *PanicHandler) Recover() {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}
		stack := debug.Stack()
		ph.errorHandler(err, stack)

		if ph.dumpToFile {
			report := CrashReport{
				Timestamp: time.Now(),
				Error:     err.Error(),
				Stack:     string(stack),
				Metadata:  ph.metadata,
			}

			if ph.includeSystemInfo {
				report.SystemInfo = SystemInfo{
					OS:           runtime.GOOS,
					Architecture: runtime.GOARCH,
					GoVersion:    runtime.Version(),
				}
			}

			ph.appendCrashReport(report)
		}

		if ph.exitOnPanic {
			os.Exit(1)
		}
	}
}

func (ph *PanicHandler) appendCrashReport(report CrashReport) {
	var reports []CrashReport

	data, err := os.ReadFile(ph.filePath)
	if err == nil {
		err := json.Unmarshal(data, &reports)
		if err != nil {
			fmt.Printf("Error unmarshalling crash reports: %v\n", err)
		}
	}

	reports = append(reports, report)

	data, _ = json.MarshalIndent(reports, "", "  ")
	err = os.WriteFile(ph.filePath, data, 0644)
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

// SetErrorHandler allows setting a custom error handler after initialization
func (ph *PanicHandler) SetErrorHandler(handler ErrorHandler) {
	ph.errorHandler = handler
}

// GetLastNCrashReports retrieves the last N crash reports from the log file
func (ph *PanicHandler) GetLastNCrashReports(n int) ([]CrashReport, error) {
	data, err := os.ReadFile(ph.filePath)
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
	return os.WriteFile(ph.filePath, []byte("[]"), 0644)
}
