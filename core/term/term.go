package term

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
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

	hostKeyCallback, err := hostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("failed to setup host key verification: %v", err)
	}

	config := &ssh.ClientConfig{
		User:            opt.Username,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback,
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

// hostKeyCallback returns an ssh.HostKeyCallback that uses TOFU (Trust On First Use).
// It reads ~/.ssh/known_hosts for verification; if the host is unknown, it appends
// the key automatically so subsequent connections are verified.
func hostKeyCallback() (ssh.HostKeyCallback, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")

	// Ensure the file exists
	if _, err := os.Stat(knownHostsPath); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(knownHostsPath, nil, 0600); err != nil {
			return nil, err
		}
	}

	cb, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, err
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := cb(hostname, remote, key)
		if err == nil {
			return nil
		}

		// If the error is "key is unknown", trust on first use and append it
		if keyErr, ok := errors.AsType[*knownhosts.KeyError](err); ok && len(keyErr.Want) == 0 {
			// No existing entry — append the new host key
			f, ferr := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY, 0600)
			if ferr != nil {
				return ferr
			}
			defer f.Close()
			line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)
			if _, ferr = fmt.Fprintln(f, line); ferr != nil {
				return ferr
			}
			return nil
		}

		// Key mismatch — possible MITM, reject
		return err
	}, nil
}
