package utils

import "testing"

func TestNormalizeAIBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no version suffix",
			input:    "https://api.openai.com",
			expected: "https://api.openai.com/v1",
		},
		{
			name:     "with trailing slash",
			input:    "https://api.openai.com/",
			expected: "https://api.openai.com/v1",
		},
		{
			name:     "already has v1",
			input:    "https://api.openai.com/v1",
			expected: "https://api.openai.com/v1",
		},
		{
			name:     "already has v1 with trailing slash",
			input:    "https://api.openai.com/v1/",
			expected: "https://api.openai.com/v1",
		},
		{
			name:     "has v2",
			input:    "https://api.openai.com/v2",
			expected: "https://api.openai.com/v2",
		},
		{
			name:     "has v3 with trailing slash",
			input:    "https://api.openai.com/v3/",
			expected: "https://api.openai.com/v3",
		},
		{
			name:     "has v10",
			input:    "https://api.openai.com/v10",
			expected: "https://api.openai.com/v10",
		},
		{
			name:     "has non-version path",
			input:    "https://api.openai.com/api",
			expected: "https://api.openai.com/api/v1",
		},
		{
			name:     "complex path without version",
			input:    "https://api.openai.com/chat/completions",
			expected: "https://api.openai.com/chat/completions/v1",
		},
		{
			name:     "complex path with version",
			input:    "https://api.openai.com/chat/v2",
			expected: "https://api.openai.com/chat/v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeAIBaseURL(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeAIBaseURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeAIBaseURLForRuntime(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		inContainer bool
		gateway     string
		expected    string
	}{
		{"container localhost", "http://localhost:8000/v1", true, "host.docker.internal", "http://host.docker.internal:8000/v1"},
		{"container ipv4 loopback", "http://127.0.0.1:8000", true, "host.docker.internal", "http://host.docker.internal:8000/v1"},
		{"container ipv6 loopback", "http://[::1]:8000/v1/", true, "gateway.internal", "http://gateway.internal:8000/v1"},
		{"native loopback unchanged", "http://127.0.0.1:8000/v1", false, "host.docker.internal", "http://127.0.0.1:8000/v1"},
		{"remote unchanged", "https://gpt.example.com/v1", true, "host.docker.internal", "https://gpt.example.com/v1"},
		{"empty", "", true, "host.docker.internal", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeAIBaseURLForRuntime(tt.input, tt.inContainer, tt.gateway); got != tt.expected {
				t.Fatalf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRewriteLoopbackURLForRuntimePreservesMediaPath(t *testing.T) {
	input := "http://127.0.0.1:8000/v1/media/images/test.png?download=1"
	want := "http://host.docker.internal:8000/v1/media/images/test.png?download=1"
	if got := RewriteLoopbackURLForRuntime(input, true, "host.docker.internal"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
