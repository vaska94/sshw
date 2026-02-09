package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/yinheli/sshw"
)

const prev = "-parent-"

var (
	Build = "devel"
	V     = flag.Bool("version", false, "show version")
	H     = flag.Bool("help", false, "show help")
	S     = flag.Bool("s", false, "use local ssh config '~/.ssh/config'")
	CopyID = flag.Bool("copy-id", false, "copy SSH public key to selected host")

	log = sshw.GetLogger()

)

func findAlias(nodes []*sshw.Node, nodeAlias string) *sshw.Node {
	for _, node := range nodes {
		if node.Alias == nodeAlias {
			return node
		}
		if len(node.Children) > 0 {
			if result := findAlias(node.Children, nodeAlias); result != nil {
				return result
			}
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if !flag.Parsed() {
		flag.Usage()
		return
	}

	if *H {
		flag.Usage()
		return
	}

	if *V {
		fmt.Println("sshw - ssh client wrapper for automatic login")
		fmt.Println("  git version:", Build)
		fmt.Println("  go version :", runtime.Version())
		return
	}
	if *S {
		err := sshw.LoadSshConfig()
		if err != nil {
			log.Error("load ssh config error", err)
			os.Exit(1)
		}
	} else {
		err := sshw.LoadConfig()
		if err != nil {
			log.Error("load config error", err)
			os.Exit(1)
		}
	}

	// login by alias
	if len(os.Args) > 1 {
		var nodeAlias = os.Args[1]
		var nodes = sshw.GetConfig()
		var node = findAlias(nodes, nodeAlias)
		if node != nil {
			client := sshw.NewClient(node)
			client.Login()
			return
		}
	}

	node := choose(nil, sshw.GetConfig())
	if node == nil {
		return
	}

	if *CopyID {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Error("cannot find home directory:", err)
			os.Exit(1)
		}
		pubKey, err := os.ReadFile(filepath.Join(home, ".ssh", "id_rsa.pub"))
		if err != nil {
			pubKey, err = os.ReadFile(filepath.Join(home, ".ssh", "id_ed25519.pub"))
			if err != nil {
				log.Error("no public key found (~/.ssh/id_rsa.pub or ~/.ssh/id_ed25519.pub)")
				os.Exit(1)
			}
		}
		client := sshw.NewClient(node)
		if err := client.CopyID(pubKey); err != nil {
			log.Error("copy-id failed:", err)
			os.Exit(1)
		}
		user := node.User
		if user == "" {
			user = "root"
		}
		fmt.Printf("public key copied to %s@%s\n", user, node.Host)
		return
	}

	client := sshw.NewClient(node)
	client.Login()
}

func choose(parent, trees []*sshw.Node) *sshw.Node {
	index, err := selectNode("select host", trees, 20)
	if err != nil || index < 0 {
		return nil
	}

	node := trees[index]
	if len(node.Children) > 0 {
		first := node.Children[0]
		if first.Name != prev {
			first = &sshw.Node{Name: prev}
			node.Children = append(node.Children[:0], append([]*sshw.Node{first}, node.Children...)...)
		}
		return choose(trees, node.Children)
	}

	if node.Name == prev {
		if parent == nil {
			return choose(nil, sshw.GetConfig())
		}
		return choose(nil, parent)
	}

	return node
}
