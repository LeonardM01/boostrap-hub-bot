package openai

import (
	"testing"
)

func TestExtractPoints(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{
			name:     "Simple number",
			input:    "5",
			expected: 5,
			hasError: false,
		},
		{
			name:     "Number with text",
			input:    "The difficulty is 7 out of 10",
			expected: 7,
			hasError: false,
		},
		{
			name:     "Number at end",
			input:    "Difficulty: 8",
			expected: 8,
			hasError: false,
		},
		{
			name:     "Two-digit number",
			input:    "10",
			expected: 10,
			hasError: false,
		},
		{
			name:     "No number",
			input:    "Very difficult",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Multiple numbers (takes first)",
			input:    "Between 3 and 8",
			expected: 3,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractPoints(tt.input)

			if tt.hasError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	// Test with empty API key
	client := New("")
	if client != nil {
		t.Error("Expected nil client with empty API key")
	}

	// Test with API key
	client = New("test-key")
	if client == nil {
		t.Error("Expected non-nil client with API key")
	}
}

func TestCalculatePointsWithoutAPIKey(t *testing.T) {
	// Create client without API key
	client := New("")

	// Should return default points
	points, err := client.CalculatePoints("Test task", "Test description")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if points != 5 {
		t.Errorf("Expected default points of 5, got %d", points)
	}
}
