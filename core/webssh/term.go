package webssh

import (
	"fmt"
	"github.com/rs/xid"
	"golang.org/x/crypto/ssh"
	"io"
	"time"
)

type TermLink struct {
	conn *ssh.Client // ssh客户端连接的实例
	Host string      // 主机地址
	Port int         // 端口号
	User string      // 用户名
}

// Dial函数使用给定的用户名和密码连接到远程主机
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
	t.conn = c    // 将ssh客户端连接实例存储到TermLink结构体属性中
	t.User = user // 存储用户名
	return nil
}

// Close方法关闭ssh客户端连接
func (t *TermLink) Close() {
	t.conn.Close()
}

// NewTerm方法创建新的终端并返回其指针
func (t *TermLink) NewTerm(rows, cols int) (*Term, error) {
	s, err := t.conn.NewSession() // 创建ssh会话
	if err != nil {
		return nil, err
	}
	stdout, err := s.StdoutPipe() // 获取远程终端的标准输出
	if err != nil {
		s.Close()
		return nil, err
	}

	stderr, err := s.StderrPipe() // 获取远程终端的标准错误输出
	if err != nil {
		s.Close()
		return nil, err
	}

	stdin, err := s.StdinPipe() // 获取远程终端的标准输入
	if err != nil {
		s.Close()
		return nil, err
	}

	// 请求一个伪终端
	err = s.RequestPty("xterm", rows, cols, ssh.TerminalModes{
		ssh.ECHO: 1, // 禁用回显
	})
	if err != nil {
		stdin.Close()
		s.Close()
		return nil, err
	}

	// 启动远程shell
	err = s.Shell()
	if err != nil {
		stdin.Close()
		s.Close()
		return nil, err
	}

	// 返回Term实例
	return &Term{
		Id:     xid.New().String(), // 使用xid库生成唯一id作为实例id
		Type:   "xterm",            // 终端类型为xterm
		Rows:   rows,
		Cols:   cols,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		s:      s,          // ssh会话实例
		t:      t,          // TermLink实例
		Since:  time.Now(), // 记录实例创建时间
	}, nil
}

type TermOption struct {
	Host       string // 主机地址
	Port       int    // 端口号
	Username   string // 用户名
	Password   string // 密码
	Rows, Cols int    // 终端窗口行数、列数
}

type Term struct {
	s      *ssh.Session   // ssh会话实例
	Id     string         `json:"id"`   // 终端实例id
	Type   string         `json:"type"` // 终端类型
	Rows   int            `json:"rows"` // 终端窗口行数
	Cols   int            `json:"cols"` // 终端窗口列数
	Stdin  io.WriteCloser `json:"-"`    // 写入到远程终端的标准输入
	Stdout io.Reader      `json:"-"`    // 从远程终端的标准输出读取数据
	Stderr io.Reader      `json:"-"`    // 从远程终端的标准错误输出读取数据
	t      *TermLink      // TermLink实例
	Since  time.Time      `json:"since"` // 终端实例创建时间
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
