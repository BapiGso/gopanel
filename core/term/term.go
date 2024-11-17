package term

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Term struct {
	Id      string         `json:"id"`
	Rows    int            `json:"rows"`
	Cols    int            `json:"cols"`
	Stdin   io.WriteCloser `json:"-"`
	Stdout  io.Reader      `json:"-"`
	Stderr  io.Reader      `json:"-"`
	session *ssh.Session
	client  *ssh.Client
	Since   time.Time `json:"since"`
}

type TermOption struct {
	Host     string
	Port     int
	Username string
	Password string
	Rows     int
	Cols     int
}

func NewTerm(opt TermOption) (*Term, error) {
	var auth []ssh.AuthMethod

	if opt.Password != "" {
		auth = append(auth, ssh.Password(opt.Password))
	}

	// 尝试读取默认的私钥文件
	homeDir, err := os.UserHomeDir()
	if err == nil {
		keyFiles := []string{"id_rsa", "id_ecdsa", "id_ed25519", "id_dsa"}
		for _, keyFile := range keyFiles {
			keyPath := filepath.Join(homeDir, ".ssh", keyFile)
			key, err := os.ReadFile(keyPath)
			if err == nil {
				signer, err := ssh.ParsePrivateKey(key)
				if err == nil {
					auth = append(auth, ssh.PublicKeys(signer))
					fmt.Printf("Added public key authentication method using %s\n", keyFile)
					break // 成功添加一个私钥后就退出循环
				} else {
					fmt.Printf("Failed to parse private key %s: %v\n", keyFile, err)
				}
			} else {
				fmt.Printf("Failed to read private key file %s: %v\n", keyFile, err)
			}
		}
	} else {
		fmt.Printf("Failed to get user home directory: %v\n", err)
	}

	config := &ssh.ClientConfig{
		User:            opt.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	// 处理 IPv6 地址
	var addr string
	if ip := net.ParseIP(opt.Host); ip != nil && ip.To4() == nil {
		// 如果是 IPv6 地址，用方括号括起来
		addr = fmt.Sprintf("[%s]:%d", opt.Host, opt.Port)
	} else {
		// IPv4 地址或主机名
		addr = fmt.Sprintf("%s:%d", opt.Host, opt.Port)
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %v", err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, err
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, err
	}

	if err := session.RequestPty("xterm", opt.Rows, opt.Cols, ssh.TerminalModes{
		ssh.ECHO: 1,
	}); err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, err
	}

	if err := session.Shell(); err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, err
	}

	return &Term{
		Id:      strconv.FormatInt(time.Now().UnixNano(), 10),
		Rows:    opt.Rows,
		Cols:    opt.Cols,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		session: session,
		client:  client,
		Since:   time.Now(),
	}, nil
}

func (t *Term) Close() {
	t.Stdin.Close()
	t.session.Close()
	t.client.Close()
}

func (t *Term) SetWindowSize(rows, cols int) error {
	err := t.session.WindowChange(rows, cols)
	if err != nil {
		return err
	}
	t.Rows = rows
	t.Cols = cols
	return nil
}
