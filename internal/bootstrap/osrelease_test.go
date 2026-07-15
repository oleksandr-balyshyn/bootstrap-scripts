package bootstrap

import (
	"strings"
	"testing"
)

func TestCheckUbuntuFrom(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "ubuntu",
			content: "ID=ubuntu\nPRETTY_NAME=\"Ubuntu 24.04 LTS\"\n",
			wantErr: false,
		},
		{
			name:    "pop os is ubuntu-like",
			content: "ID=pop\nID_LIKE=\"ubuntu debian\"\nPRETTY_NAME=\"Pop!_OS\"\n",
			wantErr: false,
		},
		{
			name:    "debian derivative",
			content: "ID=linuxmint\nID_LIKE=debian\n",
			wantErr: false,
		},
		{
			name:    "fedora is rejected",
			content: "ID=fedora\nPRETTY_NAME=\"Fedora Linux 40\"\n",
			wantErr: true,
		},
		{
			name:    "arch is rejected",
			content: "ID=arch\nID_LIKE=\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkUbuntuFrom(strings.NewReader(tt.content))
			if tt.wantErr && err == nil {
				t.Fatal("checkUbuntuFrom() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("checkUbuntuFrom() error = %v, want nil", err)
			}
		})
	}
}

func TestCheckUbuntuErrorMentionsOverride(t *testing.T) {
	err := checkUbuntuFrom(strings.NewReader("ID=fedora\nPRETTY_NAME=\"Fedora Linux 40\"\n"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--skip-os-check") {
		t.Fatalf("error should mention the override flag: %v", err)
	}
	if !strings.Contains(err.Error(), "Fedora") {
		t.Fatalf("error should name the detected system: %v", err)
	}
}
