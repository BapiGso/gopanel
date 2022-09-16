package main

import (
	"errors"
	"fmt"
	"github.com/rs/xid"
	"golang.org/x/crypto/ssh"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

type TermRef struct {
	term   *Term
	refcnt int
}

type TermStore struct {
	sync.Mutex
	terms map[string]*TermRef
}

var termStore = &TermStore{
	terms: map[string]*TermRef{},
}

func (store *TermStore) All() []*Term {
	store.Lock()
	defer store.Unlock()

	terms := []*Term{}
	for _, t := range store.terms {
		terms = append(terms, t.term)
	}

	return terms
}

func (store *TermStore) New(o TermOption) (*Term, error) {
	if o.Port == 0 {
		o.Port = 22
	}

	l := &TermLink{Host: o.Host, Port: o.Port}

	err := l.Dial(o.Username, o.Password)
	if err != nil {
		return nil, err
	}

	if o.Rows == 0 || o.Cols == 0 {
		o.Rows = 40
		o.Cols = 80
	}

	term, err := l.NewTerm(o.Rows, o.Cols)
	if err != nil {
		l.Close()
		return nil, err
	}

	store.Lock()
	defer store.Unlock()

	store.terms[term.Id] = &TermRef{
		term:   term,
		refcnt: 1, // initial refcnt is 1, call put to release.
	}

	return term, nil
}

// Lookup do not increment refcnt!
func (store *TermStore) Lookup(id string) (*Term, error) {
	store.Lock()
	defer store.Unlock()

	r, okay := store.terms[id]
	if !okay {
		return nil, errors.New("term " + id + " not exist")
	}

	return r.term, nil
}

func (store *TermStore) Get(id string) (*Term, error) {
	store.Lock()
	defer store.Unlock()

	r, okay := store.terms[id]
	if !okay {
		return nil, errors.New("term " + id + " not exist")
	}

	r.refcnt += 1

	return r.term, nil
}

func (store *TermStore) Put(id string) {
	store.Lock()
	defer store.Unlock()

	r, okay := store.terms[id]
	if !okay {
		return
	}

	r.refcnt -= 1

	if r.refcnt == 0 {
		delete(store.terms, id)
		r.term.Close()
	}
}

func (store *TermStore) Do(id string, f func(term *Term) error) error {
	store.Lock()
	defer store.Unlock()

	r, ok := store.terms[id]
	if !ok {
		return errors.New("term " + id + " not exist")
	}

	return f(r.term)
}

func listTermHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "index.template", termStore.All())
}

type newTermReq struct {
	Host     string `query:"host" form:"host" json:"host"`
	Port     int    `query:"port" form:"port" json:"port"`
	Username string `query:"user" form:"user" json:"user"`
	Password string `query:"pwd" form:"pwd" json:"pwd"`
	Rows     int    `query:"rows" form:"rows" json:"rows"`
	Cols     int    `query:"cols" form:"cols" json:"cols"`
}

func newTermHandler(c echo.Context) error {
	req := new(newTermReq)

	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.Host == "" {
		return echo.NewHTTPError(http.StatusBadRequest,
			"Host not provided")
	}

	if req.Username == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest,
			"User or password not provided")
	}

	term, err := termStore.New(TermOption{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	c.Logger().Infof("Created term: %s", term)

	return c.Render(http.StatusOK, "term.template", term)
}

type termErr struct {
	Cause string `json:"cause"`
}

func createTermHandler(c echo.Context) error {
	req := new(newTermReq)

	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}

	if req.Host == "" {
		return c.JSON(http.StatusBadRequest, termErr{"Host not provided"})
	}

	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, termErr{"User or password not provided"})
	}

	term, err := termStore.New(TermOption{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}

	c.Logger().Infof("Created term: %s", term)

	return c.JSON(http.StatusOK, term)
}

type setTermWindowSizeReq struct {
	Rows int `query:"rows" form:"rows" json:"rows"`
	Cols int `query:"cols" form:"cols" json:"cols"`
}

func setTermWindowSizeHandler(c echo.Context) error {
	req := new(setTermWindowSizeReq)

	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}

	if req.Rows == 0 || req.Cols == 0 {
		return c.JSON(http.StatusBadRequest, termErr{"Rows or cols can't be zero"})
	}

	term, err := termStore.Get(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}
	defer termStore.Put(term.Id)

	err = term.SetWindowSize(req.Rows, req.Cols)
	if err != nil {
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}

	return c.JSON(http.StatusOK, term)
}

const TermBufferSize = 8192

func linkTermDataHandler(c echo.Context) error {
	term, err := termStore.Lookup(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}

	websocket.Handler(func(ws *websocket.Conn) {
		defer func() {
			c.Logger().Infof("Destroy term: %s", term)
			termStore.Put(term.Id)
			ws.Close()
		}()

		c.Logger().Infof("Linking term: %s", term)

		go func() {
			b := [TermBufferSize]byte{}
			for {
				n, err := term.Stdout.Read(b[:])
				if err != nil {
					if !errors.Is(err, io.EOF) {
						websocket.Message.Send(ws,
							fmt.Sprintf("\nError: %s", err.Error()))
						c.Logger().Error(err)
					}
					return
				}
				if n == 0 {
					continue
				}
				websocket.Message.Send(ws, string(b[:n]))
			}
		}()

		go func() {
			b := [TermBufferSize]byte{}
			for {
				n, err := term.Stderr.Read(b[:])
				if err != nil {
					if !errors.Is(err, io.EOF) {
						websocket.Message.Send(ws,
							fmt.Sprintf("\nError: %s", err.Error()))
						c.Logger().Error(err)
					}
					return
				}
				if n == 0 {
					continue
				}
				websocket.Message.Send(ws, string(b[:n]))
			}
		}()

		for {
			b := ""
			err := websocket.Message.Receive(ws, &b)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					c.Logger().Error(err)
				}
				return
			}
			_, err = term.Stdin.Write([]byte(b))
			if err != nil {
				if !errors.Is(err, io.EOF) {
					websocket.Message.Send(ws,
						fmt.Sprintf("\nError: %s", err.Error()))
					c.Logger().Error(err)
				}
				return
			}
		}
	}).ServeHTTP(c.Response(), c.Request())

	return nil
}

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
