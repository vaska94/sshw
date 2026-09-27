package sshw

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/kevinburke/ssh_config"
	"go.yaml.in/yaml/v3"
)

type Node struct {
	Name           string           `yaml:"name"`
	Alias          string           `yaml:"alias"`
	Host           string           `yaml:"host"`
	User           string           `yaml:"user"`
	Port           int              `yaml:"port"`
	KeyPath        string           `yaml:"keypath"`
	AgentPath      string           `yaml:"agentpath"`
	Passphrase     string           `yaml:"passphrase"`
	Password       string           `yaml:"password"`
	CallbackShells []*CallbackShell `yaml:"callback-shells"`
	Children       []*Node          `yaml:"children"`
	Jump           []*Node          `yaml:"jump"`
}

type CallbackShell struct {
	Cmd   string        `yaml:"cmd"`
	Delay time.Duration `yaml:"delay"`
}

// UnmarshalYAML accepts delay as a number of milliseconds (1500)
// or as a duration string (1.5s).
func (s *CallbackShell) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Cmd   string    `yaml:"cmd"`
		Delay yaml.Node `yaml:"delay"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	s.Cmd = raw.Cmd
	s.Delay = 0
	if raw.Delay.Kind == 0 || raw.Delay.Tag == "!!null" {
		return nil
	}
	if raw.Delay.Kind != yaml.ScalarNode {
		return fmt.Errorf("line %d: delay must be milliseconds or a duration like 1.5s", raw.Delay.Line)
	}
	if ms, err := strconv.ParseInt(raw.Delay.Value, 10, 64); err == nil {
		s.Delay = time.Duration(ms) * time.Millisecond
	} else if d, err := time.ParseDuration(raw.Delay.Value); err == nil {
		s.Delay = d
	} else {
		return fmt.Errorf("line %d: invalid delay %q: use milliseconds or a duration like 1.5s", raw.Delay.Line, raw.Delay.Value)
	}
	if s.Delay < 0 {
		return fmt.Errorf("line %d: delay must not be negative", raw.Delay.Line)
	}
	return nil
}

func (n *Node) String() string {
	return n.Name
}

func (n *Node) user() string {
	if n.User == "" {
		return "root"
	}
	return n.User
}

func (n *Node) port() int {
	if n.Port <= 0 {
		return 22
	}
	return n.Port
}

var (
	config []*Node
)

func GetConfig() []*Node {
	return config
}

func LoadConfig() error {
	b, err := LoadConfigBytes(".sshw", ".sshw.yml", ".sshw.yaml")
	if err != nil {
		return err
	}
	var c []*Node
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return err
	}

	config = c

	return nil
}

func LoadSshConfig() error {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	f, err := os.Open(filepath.Join(u.HomeDir, ".ssh/config"))
	if err != nil {
		return fmt.Errorf("open ssh config: %w", err)
	}
	defer f.Close()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return fmt.Errorf("decode ssh config: %w", err)
	}
	var nc []*Node
	for _, host := range cfg.Hosts {
		alias := host.Patterns[0].String()
		hostName, err := cfg.Get(alias, "HostName")
		if err != nil {
			return err
		}
		if hostName != "" {
			port, _ := cfg.Get(alias, "Port")
			if port == "" {
				port = "22"
			}
			var c = new(Node)
			c.Name = alias
			c.Alias = alias
			c.Host = hostName
			c.User, _ = cfg.Get(alias, "User")
			c.Port, _ = strconv.Atoi(port)
			keyPath, _ := cfg.Get(alias, "IdentityFile")
			c.KeyPath, _ = expandHome(keyPath)
			agentPath, _ := cfg.Get(alias, "IdentityAgent")
			c.AgentPath, _ = expandHome(agentPath)
			nc = append(nc, c)
		}
	}
	config = nc
	return nil
}

func LoadConfigBytes(names ...string) ([]byte, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	var lastErr error
	// homedir
	for i := range names {
		sshw, err := os.ReadFile(filepath.Join(u.HomeDir, names[i]))
		if err == nil {
			return sshw, nil
		}
		lastErr = err
	}
	// relative
	for i := range names {
		sshw, err := os.ReadFile(names[i])
		if err == nil {
			return sshw, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// expandHome expands a leading ~ in path to the user's home directory.
func expandHome(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}
	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return path, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, path[1:]), nil
}
