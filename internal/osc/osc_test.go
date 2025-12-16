package osc

import (
	"testing"
)

func TestBuildOSC9(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "simple message",
			message: "Hello World",
			want:    "\x1b]9;Hello World\x07",
		},
		{
			name:    "message with special chars",
			message: "Build complete!",
			want:    "\x1b]9;Build complete!\x07",
		},
		{
			name:    "empty message",
			message: "",
			want:    "\x1b]9;\x07",
		},
		{
			name:    "message with semicolon",
			message: "foo;bar",
			want:    "\x1b]9;foo:bar\x07", // semicolon sanitized to colon
		},
		{
			name:    "message with control chars",
			message: "hello\x00world",
			want:    "\x1b]9;helloworld\x07", // control chars removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildOSC9(tt.message)
			if got != tt.want {
				t.Errorf("BuildOSC9() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildOSC777(t *testing.T) {
	tests := []struct {
		name    string
		message string
		title   string
		want    string
	}{
		{
			name:    "with title",
			message: "Build complete",
			title:   "My App",
			want:    "\x1b]777;notify;My App;Build complete\x07",
		},
		{
			name:    "empty title uses default",
			message: "Hello",
			title:   "",
			want:    "\x1b]777;notify;Notification;Hello\x07",
		},
		{
			name:    "title with semicolon",
			message: "body",
			title:   "foo;bar",
			want:    "\x1b]777;notify;foo:bar;body\x07",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildOSC777(tt.message, tt.title)
			if got != tt.want {
				t.Errorf("BuildOSC777() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildOSC99(t *testing.T) {
	tests := []struct {
		name    string
		message string
		title   string
		urgency int
		id      string
		want    string
	}{
		{
			name:    "simple message no title",
			message: "Hello",
			title:   "",
			urgency: UrgencyNormal,
			id:      "",
			want:    "\x1b]99;;Hello\x1b\\",
		},
		{
			name:    "with urgency low",
			message: "Hello",
			title:   "",
			urgency: UrgencyLow,
			id:      "",
			want:    "\x1b]99;u=0;Hello\x1b\\",
		},
		{
			name:    "with urgency critical",
			message: "Alert!",
			title:   "",
			urgency: UrgencyCritical,
			id:      "",
			want:    "\x1b]99;u=2;Alert!\x1b\\",
		},
		{
			name:    "with ID",
			message: "Progress",
			title:   "",
			urgency: UrgencyNormal,
			id:      "build-123",
			want:    "\x1b]99;i=build-123;Progress\x1b\\",
		},
		{
			name:    "with ID and urgency",
			message: "Critical",
			title:   "",
			urgency: UrgencyCritical,
			id:      "alert-1",
			want:    "\x1b]99;i=alert-1:u=2;Critical\x1b\\",
		},
		{
			name:    "with title (multi-part)",
			message: "Build finished",
			title:   "My App",
			urgency: UrgencyNormal,
			id:      "",
			want:    "\x1b]99;d=0:p=title;My App\x1b\\\x1b]99;d=1:p=body;Build finished\x1b\\",
		},
		{
			name:    "with title and urgency",
			message: "Server down!",
			title:   "Alert",
			urgency: UrgencyCritical,
			id:      "",
			want:    "\x1b]99;d=0:p=title:u=2;Alert\x1b\\\x1b]99;d=1:p=body;Server down!\x1b\\",
		},
		{
			name:    "with title, ID, and urgency",
			message: "50%",
			title:   "Progress",
			urgency: UrgencyLow,
			id:      "prog",
			want:    "\x1b]99;d=0:p=title:i=prog:u=0;Progress\x1b\\\x1b]99;d=1:p=body;50%\x1b\\",
		},
		{
			name:    "urgency clamped below 0",
			message: "test",
			title:   "",
			urgency: -1,
			id:      "",
			want:    "\x1b]99;u=0;test\x1b\\",
		},
		{
			name:    "urgency clamped above 2",
			message: "test",
			title:   "",
			urgency: 5,
			id:      "",
			want:    "\x1b]99;u=2;test\x1b\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildOSC99(tt.message, tt.title, tt.urgency, tt.id)
			if got != tt.want {
				t.Errorf("BuildOSC99() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildOSC99Close(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "valid ID",
			id:   "notif-123",
			want: "\x1b]99;i=notif-123:p=close;\x1b\\",
		},
		{
			name: "empty ID",
			id:   "",
			want: "",
		},
		{
			name: "ID with invalid chars sanitized",
			id:   "foo@bar",
			want: "\x1b]99;i=foobar:p=close;\x1b\\",
		},
		{
			name: "ID becomes empty after sanitization",
			id:   "@#$%",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildOSC99Close(tt.id)
			if got != tt.want {
				t.Errorf("BuildOSC99Close() = %q, want %q", got, tt.want)
			}
		})
	}
}
