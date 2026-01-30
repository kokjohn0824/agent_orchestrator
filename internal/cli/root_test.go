package cli

import (
	"testing"
)

func TestParseDetachChild(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"no args", []string{}, false},
		{"only binary", []string{"/path/to/agent-orchestrator"}, false},
		{"work without detach-child", []string{"/path/to/agent-orchestrator", "work"}, false},
		{"with detach-child", []string{"/path/to/agent-orchestrator", "work", "--detach-child"}, true},
		{"detach-child in middle", []string{"/path/to/agent-orchestrator", "work", "--detach-child", "TICKET-001"}, true},
		{"detach-child first after binary", []string{"/path/to/agent-orchestrator", "--detach-child", "work"}, true},
		{"similar flag not matched", []string{"/path/to/agent-orchestrator", "work", "--detach-child-extra"}, false},
		{"detach-child with equals not matched", []string{"/path/to/agent-orchestrator", "work", "--detach-child=true"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseDetachChild(tt.args)
			defer func() { isDetachChild = false }()
			if got := IsDetachChild(); got != tt.want {
				t.Errorf("parseDetachChild(%v) then IsDetachChild() = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestIsDetachChild(t *testing.T) {
	orig := isDetachChild
	defer func() { isDetachChild = orig }()

	isDetachChild = false
	if got := IsDetachChild(); got != false {
		t.Errorf("IsDetachChild() = %v, want false", got)
	}
	isDetachChild = true
	if got := IsDetachChild(); got != true {
		t.Errorf("IsDetachChild() = %v, want true", got)
	}
}
