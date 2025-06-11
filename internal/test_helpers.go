package internal

import (
	"testing"
	"time"
)

// TestHelper provides common testing utilities
type TestHelper struct {
	t *testing.T
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{t: t}
}

// AssertNoError fails the test if err is not nil
func (th *TestHelper) AssertNoError(err error) {
	th.t.Helper()
	if err != nil {
		th.t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertError fails the test if err is nil
func (th *TestHelper) AssertError(err error) {
	th.t.Helper()
	if err == nil {
		th.t.Fatal("Expected an error, got nil")
	}
}

// AssertEqual fails the test if expected != actual
func (th *TestHelper) AssertEqual(expected, actual interface{}) {
	th.t.Helper()
	if expected != actual {
		th.t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotNil fails the test if value is nil
func (th *TestHelper) AssertNotNil(value interface{}) {
	th.t.Helper()
	if value == nil {
		th.t.Fatal("Expected non-nil value")
	}
}

// MockTime returns a fixed time for testing
func MockTime() time.Time {
	return time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
}