package term

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v5"
	"golang.org/x/net/websocket"
	"io"
	"net/http"
	"sync"

	"gopanel/core/httpapi"
)

var (
	activeTerms = make(map[string]*Term)
	termMutex   sync.Mutex
)

func Index(c *echo.Context) error {
	termMutex.Lock()
	terms := make([]*Term, 0, len(activeTerms))
	for _, t := range activeTerms {
		terms = append(terms, t)
	}
	termMutex.Unlock()
	return c.Render(http.StatusOK, "term.template", terms)
}

func CreateTermHandler(c *echo.Context) error {
	req := &terminalCreate{}
	if err := httpapi.Decode(c, req); err != nil {
		return err
	}
	if req.Port == 0 {
		req.Port = 22
	}
	if req.Rows == 0 {
		req.Rows = 24
	}
	if req.Cols == 0 {
		req.Cols = 80
	}
	term, err := NewTerm(TermOption{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Rows:     req.Rows,
		Cols:     req.Cols,
	})
	if err != nil {
		return httpapi.NewError(http.StatusBadGateway, "ssh_connection_failed", err.Error())
	}

	termMutex.Lock()
	activeTerms[term.Id] = term
	termMutex.Unlock()

	return httpapi.Respond(c, http.StatusCreated, term.view())
}

func SetTermWindowSizeHandler(c *echo.Context) error {
	req := &terminalResize{}
	if err := httpapi.Decode(c, req); err != nil {
		return err
	}

	termMutex.Lock()
	term, exists := activeTerms[c.Param("id")]
	termMutex.Unlock()

	if !exists {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Terminal not found")
	}

	err := term.SetWindowSize(req.Rows, req.Cols)
	if err != nil {
		return err
	}
	return httpapi.OK(c, term.view())
}

func LinkTermDataHandler(c *echo.Context) error {
	const TermBufferSize = 8192
	id := c.Param("id")

	termMutex.Lock()
	term, exists := activeTerms[id]
	termMutex.Unlock()

	if !exists {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Terminal not found")
	}

	websocket.Handler(func(ws *websocket.Conn) {
		defer func() {
			c.Logger().Info("destroy term", "id", term.Id)
			termMutex.Lock()
			delete(activeTerms, term.Id)
			termMutex.Unlock()
			term.Close()
			ws.Close()
		}()
		c.Logger().Info("linking term", "id", term.Id)
		go func() {
			b := [TermBufferSize]byte{}
			for {
				n, err := term.Stdout.Read(b[:])
				if err != nil {
					if !errors.Is(err, io.EOF) {
						websocket.Message.Send(ws,
							fmt.Sprintf("\nError: %s", err.Error()))
						c.Logger().Error(err.Error())
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
						c.Logger().Error(err.Error())
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
			msg := ""
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				c.Logger().Error(err.Error())
				return
			}
			_, err = term.Stdin.Write([]byte(msg))
			if err != nil {
				c.Logger().Error(err.Error())
				return
			}
		}
	}).ServeHTTP(c.Response(), c.Request())

	return nil
}
