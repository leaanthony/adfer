package adfer

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("Default options", func(t *testing.T) {
		ph := New(Options{})
		if ph == nil {
			t.Fatal("Expected non-nil PanicHandler")
		}
		if ph.options.ErrorHandler == nil {
			t.Error("Expected non-nil ErrorHandler")
		}
	})

	t.Run("Custom options", func(t *testing.T) {
		customHandler := func(error, []byte) {}
		ph := New(Options{
			ErrorHandler:      customHandler,
			DumpToFile:        true,
			FilePath:          "test.json",
			ExitOnPanic:       true,
			IncludeSystemInfo: true,
			Metadata:          map[string]string{"test": "value"},
			WipeFile:          true,
		})
		if ph == nil {
			t.Fatal("Expected non-nil PanicHandler")
		}
		if ph.options.ErrorHandler == nil {
			t.Error("Expected non-nil ErrorHandler")
		}
		if !ph.options.DumpToFile {
			t.Error("Expected DumpToFile to be true")
		}
		if ph.options.FilePath != "test.json" {
			t.Errorf("Expected FilePath to be 'test.json', got '%s'", ph.options.FilePath)
		}
		if !ph.options.ExitOnPanic {
			t.Error("Expected ExitOnPanic to be true")
		}
		if !ph.options.IncludeSystemInfo {
			t.Error("Expected IncludeSystemInfo to be true")
		}
		if !reflect.DeepEqual(ph.options.Metadata, map[string]string{"test": "value"}) {
			t.Error("Expected Metadata to match")
		}
		if !ph.options.WipeFile {
			t.Error("Expected WipeFile to be true")
		}
	})
}

func TestRecover(t *testing.T) {
	t.Run("No panic", func(t *testing.T) {
		ph := New(Options{})
		func() {
			defer ph.Recover()
		}()
		// If we reach here, no panic occurred
	})

	t.Run("Recover from panic", func(t *testing.T) {
		var recoveredErr error
		var recoveredStack []byte
		ph := New(Options{
			ErrorHandler: func(err error, stack []byte) {
				recoveredErr = err
				recoveredStack = stack
			},
		})
		func() {
			defer ph.Recover()
			panic("test panic")
		}()
		if recoveredErr == nil || recoveredErr.Error() != "test panic" {
			t.Errorf("Expected error 'test panic', got '%v'", recoveredErr)
		}
		if len(recoveredStack) == 0 {
			t.Error("Expected non-empty stack trace")
		}
	})
}

func TestSafeGo(t *testing.T) {
	ph := New(Options{})
	done := make(chan bool)

	ph.SafeGo(func() {
		panic("test panic")
		done <- true
	})

	select {
	case <-done:
		t.Fatal("Goroutine should have panicked")
	case <-time.After(100 * time.Millisecond):
		// Success: goroutine panicked and was recovered
	}
}

func TestGetLastNCrashReports(t *testing.T) {
	tempFile, err := os.CreateTemp("", "crash_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	reports := []CrashReport{
		{Timestamp: time.Now(), Error: "error1"},
		{Timestamp: time.Now(), Error: "error2"},
		{Timestamp: time.Now(), Error: "error3"},
	}
	data, _ := json.Marshal(reports)
	err = os.WriteFile(tempFile.Name(), data, 0644)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	ph := New(Options{FilePath: tempFile.Name()})

	t.Run("Get all reports", func(t *testing.T) {
		result, err := ph.GetLastNCrashReports(3)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("Expected 3 reports, got %d", len(result))
		}
	})

	t.Run("Get last 2 reports", func(t *testing.T) {
		result, err := ph.GetLastNCrashReports(2)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 reports, got %d", len(result))
		}
		if result[0].Error != "error2" || result[1].Error != "error3" {
			t.Errorf("Unexpected report order")
		}
	})

	t.Run("Get more reports than available", func(t *testing.T) {
		result, err := ph.GetLastNCrashReports(5)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("Expected 3 reports, got %d", len(result))
		}
	})
	t.Run("Invalid FilePath", func(t *testing.T) {
		ph := New(Options{FilePath: ""})
		_, err := ph.GetLastNCrashReports(1)
		if err == nil {
			t.Error("Expected error for invalid FilePath, got nil")
		}
		if err.Error() != "no file path set for crash reports" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		ph := New(Options{FilePath: "non_existent_file.json"})
		_, err := ph.GetLastNCrashReports(1)
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
		if !os.IsNotExist(err) {
			t.Errorf("Expected 'file not found' error, got: %v", err)
		}
	})

	t.Run("Bad JSON in file", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "bad_json_*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		badJSON := `[{"timestamp": "2023-04-01T12:00:00Z", "error": "test error", "stack": "test stack"},`
		err = os.WriteFile(tempFile.Name(), []byte(badJSON), 0644)
		if err != nil {
			t.Fatalf("Failed to write bad JSON to temp file: %v", err)
		}

		ph := New(Options{FilePath: tempFile.Name()})
		_, err = ph.GetLastNCrashReports(1)
		if err == nil {
			t.Error("Expected error for bad JSON, got nil")
		}
		if _, ok := err.(*json.SyntaxError); !ok {
			t.Errorf("Expected json.SyntaxError, got: %T", err)
		}
	})
}

func TestWipeCrashFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", "crash_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	reports := []CrashReport{{Timestamp: time.Now(), Error: "error1"}}
	data, _ := json.Marshal(reports)
	err = os.WriteFile(tempFile.Name(), data, 0644)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	ph := New(Options{FilePath: tempFile.Name()})

	err = ph.WipeCrashFile()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	data, err = os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	if string(data) != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", string(data))
	}
}

func TestWipeCrashFileOnInitialization(t *testing.T) {
	tempFile, err := os.CreateTemp("", "crash_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	reports := []CrashReport{{Timestamp: time.Now(), Error: "error1"}}
	data, _ := json.Marshal(reports)
	err = os.WriteFile(tempFile.Name(), data, 0644)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	New(Options{
		FilePath:   tempFile.Name(),
		DumpToFile: true,
		WipeFile:   true,
	})

	data, err = os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	if string(data) != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", string(data))
	}
}

func TestDumpToFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", "crash_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	ph := New(Options{
		DumpToFile:        true,
		FilePath:          tempFile.Name(),
		IncludeSystemInfo: true,
		Metadata:          map[string]string{"test": "value"},
	})

	func() {
		defer ph.Recover()
		panic("test panic")
	}()

	data, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	var reports []CrashReport
	err = json.Unmarshal(data, &reports)
	if err != nil {
		t.Fatalf("Failed to unmarshal crash reports: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("Expected 1 report, got %d", len(reports))
	}

	report := reports[0]
	if report.Error != "test panic" {
		t.Errorf("Expected error 'test panic', got '%s'", report.Error)
	}
	if report.SystemInfo.OS != runtime.GOOS {
		t.Errorf("Expected OS '%s', got '%s'", runtime.GOOS, report.SystemInfo.OS)
	}
	if report.Metadata["test"] != "value" {
		t.Errorf("Expected metadata value 'value', got '%s'", report.Metadata["test"])
	}
	if !strings.Contains(report.Stack, "panic") {
		t.Error("Expected stack trace to contain 'panic'")
	}
}

func TestExitOnPanic(t *testing.T) {
	if os.Getenv("TEST_EXIT") == "1" {
		ph := New(Options{ExitOnPanic: true})
		defer ph.Recover()
		panic("test panic")
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExitOnPanic")
	cmd.Env = append(os.Environ(), "TEST_EXIT=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return // test passed
	}
	t.Fatalf("Process ran with err %v, want exit status 1", err)
}

func TestWipeCrashFileInvalidPath(t *testing.T) {
	// Create a PanicHandler with an invalid FilePath
	ph := New(Options{
		FilePath: "", // Empty string as an invalid path
	})

	// Attempt to wipe the crash file
	err := ph.WipeCrashFile()

	// Check if an error was returned
	if err == nil {
		t.Error("Expected an error for invalid FilePath, but got nil")
	}

	// Check if the error message is correct
	expectedErr := "no file path set for crash reports"
	if err.Error() != expectedErr {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErr, err.Error())
	}
}

func TestNewErrorOnWipeCrashFile(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "adfer_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file that can't be written to
	filePath := filepath.Join(tempDir, "crash_report.json")
	err = os.WriteFile(filePath, []byte("[]"), 0444) // Read-only file
	if err != nil {
		t.Fatalf("Failed to create read-only file: %v", err)
	}

	// Redirect stdout to capture the error message
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a new PanicHandler with options that should cause WipeCrashFile to fail
	New(Options{
		DumpToFile: true,
		FilePath:   filePath,
		WipeFile:   true,
	})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check if the error message was printed
	expectedError := "Error wiping crash file:"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message containing '%s', but got: %s", expectedError, output)
	}
}

func TestRecoverWithExitOnPanic(t *testing.T) {
	exitCalled := false
	ph := New(Options{
		ExitOnPanic: true,
	})
	ph.exitFunc = func(code int) {
		exitCalled = true
		if code != 1 {
			t.Errorf("Expected exit code 1, got %d", code)
		}
	}

	func() {
		defer ph.Recover()
		panic("test panic")
	}()

	if !exitCalled {
		t.Error("Expected exit function to be called, but it wasn't")
	}
}

func TestAppendCrashReportWriteError(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "adfer_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a read-only file
	filePath := filepath.Join(tempDir, "crash_report.json")
	err = os.WriteFile(filePath, []byte("[]"), 0444) // Read-only file
	if err != nil {
		t.Fatalf("Failed to create read-only file: %v", err)
	}

	// Create a PanicHandler with the read-only file
	ph := New(Options{
		DumpToFile: true,
		FilePath:   filePath,
	})

	// Redirect stdout to capture the error message
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Attempt to append a crash report
	ph.appendCrashReport(CrashReport{
		Timestamp: time.Now(),
		Error:     "test error",
		Stack:     "test stack",
	})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check if the error message was printed
	expectedError := "Error writing crash report to file:"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message containing '%s', but got: %s", expectedError, output)
	}
}
