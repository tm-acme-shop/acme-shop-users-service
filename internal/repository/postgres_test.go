package repository

import (
	"testing"
)

func TestGenerateUserID(t *testing.T) {
	id := generateUserID()

	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	if len(id) < 10 {
		t.Fatal("expected ID length >= 10")
	}

	// ID should start with "user-"
	if id[:5] != "user-" {
		t.Fatalf("expected ID to start with 'user-', got %s", id)
	}
}

func TestRandomString(t *testing.T) {
	tests := []struct {
		length int
	}{
		{0},
		{5},
		{10},
		{20},
	}

	for _, tt := range tests {
		result := randomString(tt.length)
		if len(result) != tt.length {
			t.Fatalf("expected length %d, got %d", tt.length, len(result))
		}
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		input    []string
		sep      string
		expected string
	}{
		{[]string{}, ", ", ""},
		{[]string{"a"}, ", ", "a"},
		{[]string{"a", "b"}, ", ", "a, b"},
		{[]string{"a", "b", "c"}, " AND ", "a AND b AND c"},
	}

	for _, tt := range tests {
		result := joinStrings(tt.input, tt.sep)
		if result != tt.expected {
			t.Fatalf("expected %q, got %q", tt.expected, result)
		}
	}
}

func TestParseName(t *testing.T) {
	// TODO(TEAM-API): Remove after v1 API deprecation
	tests := []struct {
		input     string
		firstName string
		lastName  string
	}{
		{"", "", ""},
		{"John", "John", ""},
		{"John Doe", "John", "Doe"},
		{"John Van Doe", "John", "Van Doe"},
	}

	for _, tt := range tests {
		firstName, lastName := parseName(tt.input)
		if firstName != tt.firstName || lastName != tt.lastName {
			t.Fatalf("parseName(%q) = (%q, %q), expected (%q, %q)",
				tt.input, firstName, lastName, tt.firstName, tt.lastName)
		}
	}
}

func TestSplitString(t *testing.T) {
	tests := []struct {
		input    string
		sep      string
		expected []string
	}{
		{"", " ", []string{}},
		{"a", " ", []string{"a"}},
		{"a b", " ", []string{"a", "b"}},
		{"a b c", " ", []string{"a", "b", "c"}},
		{"a  b", " ", []string{"a", "b"}}, // Multiple spaces
	}

	for _, tt := range tests {
		result := splitString(tt.input, tt.sep)
		if len(result) != len(tt.expected) {
			t.Fatalf("splitString(%q, %q) = %v, expected %v",
				tt.input, tt.sep, result, tt.expected)
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Fatalf("splitString(%q, %q) = %v, expected %v",
					tt.input, tt.sep, result, tt.expected)
			}
		}
	}
}

func TestUserCacheKey(t *testing.T) {
	// Deprecated function test
	// TODO(TEAM-PLATFORM): Remove after cache key function removal
	key := userCacheKey("user-123")
	expected := "user:user-123"

	if key != expected {
		t.Fatalf("expected %q, got %q", expected, key)
	}
}
