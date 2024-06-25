package adfer

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ph := New()
	if ph == nil {
		t.Error("New() returned nil")
	}
}

func TestWithErrorHandler(t *testing.T) {
	customHandler := func(err error, stack []byte) {}
	ph := New(WithErrorHandler(customHandler))
	if ph.errorHandler == nil {
		t.Error("WithErrorHandler() did not set the error handler")
	}
}

func TestWithDumpToFile(t *testing.T) {
	ph := New(WithDumpToFile("test.log"))
	if !ph.dumpToFile {
		t.Error("WithDumpToFile() did not enable file dumping")
	}
	if ph.filePath != "test.log" {
		t.Error("WithDumpToFile() did not set the correct file path")
	}
}

func TestWithExitOnPanic(t *testing.T) {
	ph := New(WithExitOnPanic())
	if !ph.exitOnPanic {
		t.Error("WithExitOnPanic() did not enable exiting on panic")
	}
}

func TestWithSystemInfo(t *testing.T) {
	ph := New(WithSystemInfo())
	if !ph.includeSystemInfo {
		t.Error("WithSystemInfo() did not enable including system info")
	}
}

func TestWithMetadata(t *testing.T) {
	metadata := map[string]string{"version": "1.0.0", "env": "test"}
	ph := New(WithMetadata(metadata))
	if len(ph.metadata) != 2 || ph.metadata["version"] != "1.0.0" || ph.metadata["env"] != "test" {
		t.Error("WithMetadata() did not set the metadata correctly")
	}
}

func TestWithWipeFileOnInit(t *testing.T) {
	ph := New(WithWipeFile())
	if !ph.wipeFile {
		t.Error("WithWipeFileOnInit() did not enable wiping file on init")
	}
}

func TestRecover(t *testing.T) {
	testFile := "test_recover.log"
	defer os.Remove(testFile)

	ph := New(
		WithDumpToFile(testFile),
		WithSystemInfo(),
		WithMetadata(map[string]string{"test": "value"}),
	)

	func() {
		defer ph.Recover()
		panic("test panic")
	}()

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read crash file: %v", err)
	}

	var reports []CrashReport
	err = json.Unmarshal(data, &reports)
	if err != nil {
		t.Fatalf("Failed to unmarshal crash reports: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("Expected 1 crash report, got %d", len(reports))
	}

	report := reports[0]
	if report.Error != "test panic" {
		t.Errorf("Expected error 'test panic', got '%s'", report.Error)
	}
	if report.SystemInfo.OS != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, report.SystemInfo.OS)
	}
	if report.Metadata["test"] != "value" {
		t.Errorf("Expected metadata 'test: value', got '%s'", report.Metadata["test"])
	}
}

func TestSafeGo(t *testing.T) {
	ph := New()
	done := make(chan bool)

	ph.SafeGo(func() {
		panic("test panic in goroutine")
	})

	go func() {
		time.Sleep(time.Millisecond * 100)
		done <- true
	}()

	<-done
	// If we reach here, it means the panic was caught and the goroutine didn't crash the program
}

func TestSetErrorHandler(t *testing.T) {
	ph := New()
	newHandler := func(err error, stack []byte) {}
	ph.SetErrorHandler(newHandler)
	if ph.errorHandler == nil {
		t.Error("SetErrorHandler() did not set the new error handler")
	}
}

func TestGetLastNCrashReports(t *testing.T) {
	testFile := "test_get_reports.log"
	defer os.Remove(testFile)

	ph := New(WithDumpToFile(testFile))

	// Generate 5 crash reports
	for i := 0; i < 5; i++ {
		func() {
			defer ph.Recover()
			panic(errors.New("test panic"))
		}()
	}

	reports, err := ph.GetLastNCrashReports(3)
	if err != nil {
		t.Fatalf("GetLastNCrashReports failed: %v", err)
	}

	if len(reports) != 3 {
		t.Errorf("Expected 3 reports, got %d", len(reports))
	}

	for _, report := range reports {
		if report.Error != "test panic" {
			t.Errorf("Expected error 'test panic', got '%s'", report.Error)
		}
	}
}

func TestGetLastNCrashReportsTooBig(t *testing.T) {
	testFile := "test_get_reports.log"
	defer os.Remove(testFile)

	ph := New(WithDumpToFile(testFile))

	// Generate 5 crash reports
	for i := 0; i < 5; i++ {
		func() {
			defer ph.Recover()
			panic(errors.New("test panic"))
		}()
	}

	_, err := ph.GetLastNCrashReports(16)
	if err == nil {
		t.Fatalf("GetLastNCrashReports should have failed")
	}
}

func TestWipeCrashFile(t *testing.T) {
	testFile := "test_wipe.log"
	defer os.Remove(testFile)

	ph := New(WithDumpToFile(testFile))

	// Generate a crash report
	func() {
		defer ph.Recover()
		panic("test panic")
	}()

	err := ph.WipeCrashFile()
	if err != nil {
		t.Fatalf("WipeCrashFile failed: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read crash file: %v", err)
	}

	if string(data) != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", string(data))
	}
}

func TestWipeFileOnInit(t *testing.T) {
	testFile := "test_wipe_on_init.log"
	defer os.Remove(testFile)

	// Create a file with some content
	os.WriteFile(testFile, []byte("test content"), 0644)

	New(
		WithDumpToFile(testFile),
		WithWipeFile(),
	)

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read crash file: %v", err)
	}

	if string(data) != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", string(data))
	}
}

func TestRecoverWithExitOnPanic(t *testing.T) {
	if os.Getenv("TEST_EXIT") == "1" {
		ph := New(WithExitOnPanic())
		defer ph.Recover()
		panic("test panic with exit")
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRecoverWithExitOnPanic")
	cmd.Env = append(os.Environ(), "TEST_EXIT=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Errorf("process ran with err %v, want exit status 1", err)
}

func TestDefaultErrorHandler(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := errors.New("test error")
	stack := []byte("test stack")
	defaultErrorHandler(err, stack)

	w.Close()
	os.Stdout = oldStdout

	out, _ := io.ReadAll(r)
	output := string(out)

	if !strings.Contains(output, "test error") || !strings.Contains(output, "test stack") {
		t.Errorf("defaultErrorHandler output doesn't contain expected content")
	}
}

func TestAppendCrashReport(t *testing.T) {
	testFile := "test_append.log"
	defer os.Remove(testFile)

	ph := New(WithDumpToFile(testFile))

	report := CrashReport{
		Timestamp: time.Now(),
		Error:     "test error",
		Stack:     "test stack",
	}

	ph.appendCrashReport(report)

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read crash file: %v", err)
	}

	var reports []CrashReport
	err = json.Unmarshal(data, &reports)
	if err != nil {
		t.Fatalf("Failed to unmarshal crash reports: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("Expected 1 crash report, got %d", len(reports))
	}

	if reports[0].Error != "test error" || reports[0].Stack != "test stack" {
		t.Errorf("Appended crash report doesn't match expected content")
	}
}
