package cli

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		want      Options
		wantError string
	}{
		{
			name:  "no args",
			input: nil,
			want:  Options{Command: CommandStart},
		},
		{
			name:  "tunnel names",
			input: []string{"db", "cache"},
			want:  Options{Command: CommandStart, TunnelNames: []string{"db", "cache"}},
		},
		{
			name:  "detach",
			input: []string{"--detach", "db"},
			want:  Options{Command: CommandStart, Detach: true, TunnelNames: []string{"db"}},
		},
		{
			name:  "short detach",
			input: []string{"-d"},
			want:  Options{Command: CommandStart, Detach: true},
		},
		{
			name:  "status",
			input: []string{"status"},
			want:  Options{Command: CommandStatus},
		},
		{
			name:  "version",
			input: []string{"version"},
			want:  Options{Command: CommandVersion},
		},
		{
			name:      "status with detach",
			input:     []string{"status", "--detach"},
			wantError: errStatusWithDetach.Error(),
		},
		{
			name:      "status with args",
			input:     []string{"status", "foo"},
			wantError: errStatusWithArgs.Error(),
		},
		{
			name:  "internal daemon",
			input: []string{"--internal-daemon"},
			want:  Options{Command: CommandStart, InternalDaemon: true},
		},
		{
			name:  "stop",
			input: []string{"stop"},
			want:  Options{Command: CommandStop},
		},
		{
			name:      "stop with detach",
			input:     []string{"stop", "-d"},
			wantError: errStopWithDetach.Error(),
		},
		{
			name:      "stop with args",
			input:     []string{"stop", "db"},
			wantError: errStopWithArgs.Error(),
		},
		{
			name:      "version with detach",
			input:     []string{"version", "--detach"},
			wantError: errVersionWithDetach.Error(),
		},
		{
			name:      "version with args",
			input:     []string{"version", "extra"},
			wantError: errVersionWithArgs.Error(),
		},
		{
			name:      "unknown flag",
			input:     []string{"--unknown"},
			wantError: "unknown flag: --unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantError)
				}
				if err.Error() != tt.wantError {
					t.Fatalf("expected error %q, got %q", tt.wantError, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Command != tt.want.Command {
				t.Fatalf("command mismatch: got %v want %v", got.Command, tt.want.Command)
			}
			if got.Detach != tt.want.Detach {
				t.Fatalf("detach mismatch: got %v want %v", got.Detach, tt.want.Detach)
			}
			if got.InternalDaemon != tt.want.InternalDaemon {
				t.Fatalf("internal daemon mismatch: got %v want %v", got.InternalDaemon, tt.want.InternalDaemon)
			}
			if len(got.TunnelNames) != len(tt.want.TunnelNames) {
				t.Fatalf("tunnel names length mismatch: got %d want %d", len(got.TunnelNames), len(tt.want.TunnelNames))
			}
			for i, v := range got.TunnelNames {
				if v != tt.want.TunnelNames[i] {
					t.Fatalf("tunnel name %d mismatch: got %s want %s", i, v, tt.want.TunnelNames[i])
				}
			}
		})
	}
}
