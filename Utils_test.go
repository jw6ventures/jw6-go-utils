package jw6_utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestLogLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"Debug", Debug},
		{"Info", Info},
		{"Warn", Warn},
		{"Error", Error},
		{"Fatal", Fatal},
		{"Unknown", Info}, // default case
		{"", Info},        // default case
		{"Trace", Info},   // not handled, should return default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := LogLevelFromString(tt.input)
			if result != tt.expected {
				t.Errorf("LogLevelFromString(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{Trace, "Trace"},
		{Debug, "Debug"},
		{Info, "Info"},
		{Warn, "Warn"},
		{Error, "Error"},
		{Fatal, "Fatal"},
		{LogLevel(-1), "Unknown"},  // below range
		{LogLevel(100), "Unknown"}, // above range
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("LogLevel(%d).String() = %q, want %q", tt.level, result, tt.expected)
			}
		})
	}
}

func TestUtils_Log(t *testing.T) {
	tests := []struct {
		name         string
		logLevel     LogLevel
		messageLevel LogLevel
		class        string
		method       string
		message      string
		shouldLog    bool
		hasColor     bool
		hasBanner    bool
	}{
		{
			name:         "Info log when level is Info",
			logLevel:     Info,
			messageLevel: Info,
			class:        "TestClass",
			method:       "TestMethod",
			message:      "Test message",
			shouldLog:    true,
			hasColor:     true,
			hasBanner:    false,
		},
		{
			name:         "Debug log when level is Info (should not log)",
			logLevel:     Info,
			messageLevel: Debug,
			class:        "TestClass",
			method:       "TestMethod",
			message:      "Debug message",
			shouldLog:    false,
			hasColor:     false,
			hasBanner:    false,
		},
		{
			name:         "Error log when level is Info",
			logLevel:     Info,
			messageLevel: Error,
			class:        "TestClass",
			method:       "TestMethod",
			message:      "Error message",
			shouldLog:    true,
			hasColor:     true,
			hasBanner:    false,
		},
		{
			name:         "Fatal log when level is Info",
			logLevel:     Info,
			messageLevel: Fatal,
			class:        "TestClass",
			method:       "TestMethod",
			message:      "Fatal message",
			shouldLog:    true,
			hasColor:     true,
			hasBanner:    true,
		},
		{
			name:         "Warn log when level is Info",
			logLevel:     Info,
			messageLevel: Warn,
			class:        "TestClass",
			method:       "TestMethod",
			message:      "Warning message",
			shouldLog:    true,
			hasColor:     true,
			hasBanner:    false,
		},
		{
			name:         "Trace log when level is Trace",
			logLevel:     Trace,
			messageLevel: Trace,
			class:        "TestClass",
			method:       "TestMethod",
			message:      "Trace message",
			shouldLog:    true,
			hasColor:     false,
			hasBanner:    false,
		},
		{
			name:         "Debug log when level is Debug",
			logLevel:     Debug,
			messageLevel: Debug,
			class:        "TestClass",
			method:       "TestMethod",
			message:      "Debug message",
			shouldLog:    true,
			hasColor:     false,
			hasBanner:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			utils := &Utils{LogLevel: tt.logLevel}
			utils.Log(tt.class, tt.method, tt.messageLevel, tt.message)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.shouldLog {
				if !strings.Contains(output, tt.class) {
					t.Errorf("Expected output to contain class %q, but it didn't. Output: %q", tt.class, output)
				}
				if !strings.Contains(output, tt.method) {
					t.Errorf("Expected output to contain method %q, but it didn't. Output: %q", tt.method, output)
				}
				if !strings.Contains(output, tt.message) {
					t.Errorf("Expected output to contain message %q, but it didn't. Output: %q", tt.message, output)
				}
				if !strings.Contains(output, tt.messageLevel.String()) {
					t.Errorf("Expected output to contain log level %q, but it didn't. Output: %q", tt.messageLevel.String(), output)
				}

				if tt.hasBanner {
					if !strings.Contains(output, "***") {
						t.Errorf("Expected Fatal log to have banner (***), but it didn't. Output: %q", output)
					}
				}
			} else {
				if output != "" {
					t.Errorf("Expected no output when message level is below log level, but got: %q", output)
				}
			}
		})
	}
}

func TestUtils_PrintBanner(t *testing.T) {
	tests := []struct {
		name           string
		product        string
		version        string
		copyrightYear  string
		hashes         int
		copyrightOwner string
	}{
		{
			name:           "Standard banner",
			product:        "Test Product",
			version:        "1.0.0",
			copyrightYear:  "2024",
			hashes:         3,
			copyrightOwner: "Test Owner",
		},
		{
			name:           "Long product name",
			product:        "Very Long Product Name That Should Control Width",
			version:        "2.0",
			copyrightYear:  "2024",
			hashes:         2,
			copyrightOwner: "Owner",
		},
		{
			name:           "Many hashes",
			product:        "Product",
			version:        "1.0",
			copyrightYear:  "2024",
			hashes:         10,
			copyrightOwner: "Owner",
		},
		{
			name:           "Minimal hashes",
			product:        "App",
			version:        "1.0",
			copyrightYear:  "2024",
			hashes:         1,
			copyrightOwner: "Me",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			utils := &Utils{LogLevel: Info}
			utils.PrintBanner(tt.product, tt.version, tt.copyrightYear, tt.hashes, tt.copyrightOwner)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Check that banner contains expected elements
			if !strings.Contains(output, tt.product) {
				t.Errorf("Expected banner to contain product %q, but it didn't", tt.product)
			}
			if !strings.Contains(output, tt.version) {
				t.Errorf("Expected banner to contain version %q, but it didn't", tt.version)
			}
			if !strings.Contains(output, tt.copyrightYear) {
				t.Errorf("Expected banner to contain copyright year %q, but it didn't", tt.copyrightYear)
			}
			if !strings.Contains(output, tt.copyrightOwner) {
				t.Errorf("Expected banner to contain copyright owner %q, but it didn't", tt.copyrightOwner)
			}
			if !strings.Contains(output, "#") {
				t.Errorf("Expected banner to contain hash symbols")
			}
			if !strings.Contains(output, "Version") {
				t.Errorf("Expected banner to contain 'Version' label")
			}
			if !strings.Contains(output, "Copyright") {
				t.Errorf("Expected banner to contain 'Copyright' label")
			}

			// Verify the banner has multiple lines
			lines := strings.Split(strings.TrimSpace(output), "\n")
			if len(lines) < 5 {
				t.Errorf("Expected banner to have at least 5 lines, got %d", len(lines))
			}
		})
	}
}

func TestUtils_centerInString(t *testing.T) {
	tests := []struct {
		name     string
		word     string
		spaces   int
		expected string
	}{
		{
			name:     "Even spaces",
			word:     "test",
			spaces:   10,
			expected: "   test   ",
		},
		{
			name:     "Odd spaces",
			word:     "test",
			spaces:   11,
			expected: "   test    ",
		},
		{
			name:     "Exact fit",
			word:     "test",
			spaces:   4,
			expected: "test",
		},
		{
			name:     "Single character",
			word:     "a",
			spaces:   5,
			expected: "  a  ",
		},
		{
			name:     "Large spaces",
			word:     "hi",
			spaces:   20,
			expected: "         hi         ",
		},
		{
			name:     "Minimal spaces",
			word:     "x",
			spaces:   3,
			expected: " x ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utils := Utils{LogLevel: Info}
			result := utils.centerInString(tt.word, tt.spaces)
			if result != tt.expected {
				t.Errorf("centerInString(%q, %d) = %q, want %q", tt.word, tt.spaces, result, tt.expected)
			}
			if len(result) != tt.spaces {
				t.Errorf("centerInString(%q, %d) returned string of length %d, want %d", tt.word, tt.spaces, len(result), tt.spaces)
			}
		})
	}
}

func TestUtils_Integration(t *testing.T) {
	t.Run("Complete workflow", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Create Utils with Info level
		utils := &Utils{LogLevel: LogLevelFromString("Info")}

		// Print banner
		utils.PrintBanner("Test App", "1.0.0", "2024", 3, "Test Corp")

		// Log various messages
		utils.Log("MainClass", "main", Trace, "This should not appear")
		utils.Log("MainClass", "main", Debug, "This should not appear either")
		utils.Log("MainClass", "main", Info, "Application started")
		utils.Log("MainClass", "process", Warn, "Low memory warning")
		utils.Log("MainClass", "process", Error, "Connection failed")

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Verify we got expected output
		if !strings.Contains(output, "Test App") {
			t.Error("Expected banner with Test App")
		}
		if !strings.Contains(output, "Application started") {
			t.Error("Expected Info log message")
		}
		if !strings.Contains(output, "Low memory warning") {
			t.Error("Expected Warn log message")
		}
		if !strings.Contains(output, "Connection failed") {
			t.Error("Expected Error log message")
		}
		if strings.Contains(output, "This should not appear") {
			t.Error("Debug and Trace messages should not appear when log level is Info")
		}
	})
}

func BenchmarkUtils_Log(b *testing.B) {
	utils := &Utils{LogLevel: Info}

	// Suppress output
	old := os.Stdout
	os.Stdout = nil
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utils.Log("TestClass", "TestMethod", Info, "Benchmark message")
	}
}

func BenchmarkUtils_centerInString(b *testing.B) {
	utils := Utils{LogLevel: Info}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utils.centerInString("test word", 50)
	}
}

func ExampleUtils_Log() {
	utils := &Utils{LogLevel: Info}
	utils.Log("ExampleClass", "ExampleMethod", Info, "This is an example log message")
}

func ExampleUtils_PrintBanner() {
	utils := &Utils{LogLevel: Info}
	utils.PrintBanner("My Application", "1.0.0", "2024", 3, "My Company")
}

func ExampleLogLevelFromString() {
	level := LogLevelFromString("Info")
	fmt.Println(level.String())
	// Output: Info
}

func ExampleLogLevel_String() {
	level := Info
	fmt.Println(level.String())
	// Output: Info
}
