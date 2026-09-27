package sshw

import (
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

func TestCallbackShellDelay(t *testing.T) {
	tests := []struct {
		name  string
		delay string
		want  time.Duration
	}{
		{"milliseconds", "1500", 1500 * time.Millisecond},
		{"quoted milliseconds", `"1500"`, 1500 * time.Millisecond},
		{"duration", "1.5s", 1500 * time.Millisecond},
		{"duration ms", "200ms", 200 * time.Millisecond},
		{"zero", "0", 0},
		{"null", "~", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nodes []*Node
			in := "- name: a\n  callback-shells:\n    - { delay: " + tt.delay + ", cmd: 0 }\n"
			if err := yaml.Unmarshal([]byte(in), &nodes); err != nil {
				t.Fatal(err)
			}
			cs := nodes[0].CallbackShells[0]
			if cs.Delay != tt.want {
				t.Errorf("delay = %v, want %v", cs.Delay, tt.want)
			}
			if cs.Cmd != "0" {
				t.Errorf("cmd = %q, want %q", cs.Cmd, "0")
			}
		})
	}
}

func TestCallbackShellDelayOmitted(t *testing.T) {
	var nodes []*Node
	if err := yaml.Unmarshal([]byte("- name: a\n  callback-shells:\n    - { cmd: echo 1 }\n"), &nodes); err != nil {
		t.Fatal(err)
	}
	if cs := nodes[0].CallbackShells[0]; cs.Delay != 0 || cs.Cmd != "echo 1" {
		t.Errorf("got %+v", cs)
	}
}

func TestCallbackShellDelayInvalid(t *testing.T) {
	for _, delay := range []string{"soon", "-5", "-1s", "[1]", "1.5"} {
		var nodes []*Node
		in := "- name: a\n  callback-shells:\n    - { delay: " + delay + " }\n"
		err := yaml.Unmarshal([]byte(in), &nodes)
		if err == nil || !strings.Contains(err.Error(), "delay") {
			t.Errorf("delay %s: err = %v, want delay error", delay, err)
		}
	}
}

func TestReadmeConfig(t *testing.T) {
	in := `
- name: dev server fully configured
  user: appuser
  host: 192.168.8.35
  port: 22
  password: 123456
  callback-shells:
    - { cmd: 2 }
    - { delay: 1500, cmd: 0 }
    - { cmd: "echo 1" }
  children:
    - name: child
      jump:
        - { name: j, host: 10.0.0.1 }
`
	var nodes []*Node
	if err := yaml.Unmarshal([]byte(in), &nodes); err != nil {
		t.Fatal(err)
	}
	n := nodes[0]
	if n.Password != "123456" || n.Port != 22 || len(n.CallbackShells) != 3 {
		t.Fatalf("got %+v", n)
	}
	if d := n.CallbackShells[1].Delay; d != 1500*time.Millisecond {
		t.Errorf("delay = %v", d)
	}
	if n.Children[0].Jump[0].Host != "10.0.0.1" {
		t.Errorf("jump host = %q", n.Children[0].Jump[0].Host)
	}
}
