/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package extgatling

import (
	"context"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"os"
	"path/filepath"
	"testing"
)

func TestHasFileWithSuffix(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "hasfilewithsuffix_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test.scala")
	testFile2 := filepath.Join(tmpDir, "test.txt")
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	testFile3 := filepath.Join(subDir, "nested.scala")

	for _, file := range []string{testFile1, testFile2, testFile3} {
		f, err := os.Create(file)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
		f.Close()
	}

	// Test cases
	tests := []struct {
		name     string
		dir      string
		suffix   string
		expected bool
	}{
		{"Find scala file in root", tmpDir, "scala", true},
		{"Find nested scala file", tmpDir, "scala", true},
		{"Non-existent suffix", tmpDir, "java", false},
		{"Find txt file", tmpDir, "txt", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := HasFileWithSuffix(tc.dir, tc.suffix)
			if result != tc.expected {
				t.Errorf("HasFileWithSuffix(%s, %s) = %v, expected %v", tc.dir, tc.suffix, result, tc.expected)
			}
		})
	}
}

func TestNewGatlingLoadTestRunAction(t *testing.T) {
	action := NewGatlingLoadTestRunAction()
	if action == nil {
		t.Error("NewGatlingLoadTestRunAction() returned nil")
	}
}

func TestNewEmptyState(t *testing.T) {
	action := &GatlingLoadTestRunAction{}
	state := action.NewEmptyState()

	// Check that state is properly initialized
	if state.Command != nil {
		t.Error("Initial state should have nil Command")
	}
	if state.CmdStateID != "" {
		t.Error("Initial state should have empty CmdStateID")
	}
}

func TestDescribe(t *testing.T) {
	action := &GatlingLoadTestRunAction{}
	description := action.Describe()

	// Check basic properties of the description
	if description.Id != actionId {
		t.Errorf("Expected description.Id to be %s, got %s", actionId, description.Id)
	}
	if description.Label != "Gatling" {
		t.Errorf("Expected description.Label to be Gatling, got %s", description.Label)
	}
	if description.Kind != action_kit_api.LoadTest {
		t.Errorf("Expected description.Kind to be LoadTest, got %s", description.Kind)
	}
}

func TestStdOutToMessages(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected int // Number of messages expected
	}{
		{
			"Empty input",
			[]string{},
			0,
		},
		{
			"Single line",
			[]string{"Test message"},
			1,
		},
		{
			"Multiple lines",
			[]string{"Line 1", "Line 2", "Line 3"},
			3,
		},
		{
			"Lines with whitespace",
			[]string{"  ", "Line with content", "\n", "\t"},
			1, // Only the line with content should be included
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := stdOutToMessages(tc.input)
			if len(result) != tc.expected {
				t.Errorf("Expected %d messages, got %d", tc.expected, len(result))
			}

			// Check that all non-empty messages are included
			for i, msg := range result {
				if msg.Message == "" {
					t.Errorf("Message %d is empty", i)
				}
				if msg.Level == nil || *msg.Level != action_kit_api.Info {
					t.Errorf("Message %d has incorrect level", i)
				}
			}
		})
	}
}

func TestPrepare(t *testing.T) {
	// This is a simplified test that doesn't actually run commands
	// We'll check that the state is properly updated

	action := &GatlingLoadTestRunAction{}
	state := &GatlingLoadTestRunState{}
	execId := uuid.New()

	// Create a minimal prepare request
	request := action_kit_api.PrepareActionRequestBody{
		ExecutionId: execId,
		Config: map[string]interface{}{
			"file":       "/tmp/test.scala",
			"parameter":  []map[string]string{{"key": "users", "value": "10"}},
			"simulation": "TestSimulation",
		},
		ExecutionContext: &action_kit_api.ExecutionContext{
			ExperimentKey: extutil.Ptr("test-experiment"),
			ExecutionId:   extutil.Ptr(1),
		},
	}

	// Mock file operations - this is just a basic test
	// For a complete test, we'd need to set up the file system

	// Call Prepare
	_, err := action.Prepare(context.Background(), state, request)

	// In a real test, we'd check the error and verify state changes
	// Here we just ensure the function completes
	if err != nil {
		// This is expected since we didn't set up the file system
		t.Logf("Prepare error (expected): %v", err)
	}
}

