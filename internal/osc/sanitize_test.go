package osc

import "testing"

func TestSanitize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text unchanged",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "semicolons replaced with colons",
			input: "foo;bar;baz",
			want:  "foo:bar:baz",
		},
		{
			name:  "null bytes removed",
			input: "hello\x00world",
			want:  "helloworld",
		},
		{
			name:  "newlines removed",
			input: "line1\nline2",
			want:  "line1line2",
		},
		{
			name:  "tabs removed",
			input: "col1\tcol2",
			want:  "col1col2",
		},
		{
			name:  "escape sequences removed",
			input: "text\x1b[31mred\x1b[0m",
			want:  "text[31mred[0m",
		},
		{
			name:  "carriage return removed",
			input: "line\r",
			want:  "line",
		},
		{
			name:  "DEL character removed",
			input: "hello\x7fworld",
			want:  "helloworld",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only control chars",
			input: "\x00\x01\x02",
			want:  "",
		},
		{
			name:  "unicode preserved",
			input: "Hello 世界 🎉",
			want:  "Hello 世界 🎉",
		},
		{
			name:  "mixed control and semicolons",
			input: "a\x00;b\nc",
			want:  "a:bc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid ID unchanged",
			input: "my-notification_123",
			want:  "my-notification_123",
		},
		{
			name:  "dots allowed",
			input: "v1.2.3",
			want:  "v1.2.3",
		},
		{
			name:  "plus sign allowed",
			input: "build+test",
			want:  "build+test",
		},
		{
			name:  "spaces removed",
			input: "my notification",
			want:  "mynotification",
		},
		{
			name:  "special chars removed",
			input: "foo@bar#baz",
			want:  "foobarbaz",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only invalid chars",
			input: "@#$%^&*()",
			want:  "",
		},
		{
			name:  "unicode removed",
			input: "notify_🎉",
			want:  "notify_",
		},
		{
			name:  "semicolons and colons removed",
			input: "a;b:c",
			want:  "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeID(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
