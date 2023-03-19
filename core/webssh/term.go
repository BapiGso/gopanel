package webssh

import (
	"fmt"
	"github.com/rs/xid"
	"golang.org/x/crypto/ssh"
	"io"
	"time"
)

type TermLink struct {
	conn *ssh.Client
	Host string
	Port int
	User string
}

func (t *TermLink) Dial(user, pwd string) error {
	c, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", t.Host, t.Port),
		&ssh.ClientConfig{
			User: user,
			Auth: []ssh.AuthMethod{
				ssh.Password(pwd),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		})
	if err != nil {
		return err
	}

	t.conn = c
	t.User = user

	return nil
}

func (t *TermLink) Close() {
	t.conn.Close()
}

func (t *TermLink) NewTerm(rows, cols int) (*Term, error) {
	s, err := t.conn.NewSession()
	if err != nil {
		return nil, err
	}

	stdout, err := s.StdoutPipe()
	if err != nil {
		s.Close()
		return nil, err
	}

	stderr, err := s.StderrPipe()
	if err != nil {
		s.Close()
		return nil, err
	}

	stdin, err := s.StdinPipe()
	if err != nil {
		s.Close()
		return nil, err
	}

	// Request pseudo terminal
	err = s.RequestPty("xterm", rows, cols, ssh.TerminalModes{
		ssh.ECHO: 1, // disable echoing
	})
	if err != nil {
		stdin.Close()
		s.Close()
		return nil, err
	}

	// Start remote shell
	err = s.Shell()
	if err != nil {
		stdin.Close()
		s.Close()
		return nil, err
	}

	return &Term{
		Id:     xid.New().String(),
		Type:   "xterm",
		Rows:   rows,
		Cols:   cols,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		s:      s,
		t:      t,
		Since:  time.Now(),
	}, nil
}

type TermOption struct {
	Host       string
	Port       int
	Username   string
	Password   string
	Rows, Cols int
}

type Term struct {
	s      *ssh.Session
	Id     string         `json:"id"`
	Type   string         `json:"type"`
	Rows   int            `json:"rows"`
	Cols   int            `json:"cols"`
	Stdin  io.WriteCloser `json:"-"`
	Stdout io.Reader      `json:"-"`
	Stderr io.Reader      `json:"-"`
	t      *TermLink
	Since  time.Time `json:"since"`
}

func (t *Term) Host() string {
	return t.t.Host
}

func (t *Term) Port() int {
	return t.t.Port
}

func (t *Term) User() string {
	return t.t.User
}

func (t *Term) Name() string {
	return fmt.Sprintf("%s@%s:%d", t.User(), t.Host(), t.Port())
}

func (t *Term) SetWindowSize(rows, cols int) error {
	err := t.s.WindowChange(rows, cols)
	if err != nil {
		return err
	}
	t.Rows = rows
	t.Cols = cols
	return nil
}

func (t *Term) String() string {
	return fmt.Sprintf("%s-%s", t.Id, t.Name())
}

func (t *Term) Close() {
	t.Stdin.Close()
	t.s.Close()
	t.t.Close()
}
