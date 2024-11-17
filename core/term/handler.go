package term

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
	"io"
	"net/http"
	"sync"
)

var (
	activeTerms = make(map[string]*Term)
	termMutex   sync.Mutex
)

func Index(c echo.Context) error {
	termMutex.Lock()
	terms := make([]*Term, 0, len(activeTerms))
	for _, t := range activeTerms {
		terms = append(terms, t)
	}
	termMutex.Unlock()
	return c.Render(http.StatusOK, "term.template", terms)
}

func CreateTermHandler(c echo.Context) error {
	req := &struct {
		Host     string `query:"host" form:"host" json:"host" validate:"required"`
		Port     int    `query:"port" form:"port" json:"port" validate:"required"`
		Username string `query:"user" form:"user" json:"user" validate:"required"`
		Password string `query:"pwd"  form:"pwd" json:"pwd"`
		Rows     int    `query:"rows" form:"rows" json:"rows"`
		Cols     int    `query:"cols" form:"cols" json:"cols"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
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
		return err
	}

	termMutex.Lock()
	activeTerms[term.Id] = term
	termMutex.Unlock()

	return c.JSON(200, term)
}

func SetTermWindowSizeHandler(c echo.Context) error {
	req := &struct {
		Id   string `query:"id" form:"id" json:"id"`
		Rows int    `query:"rows" form:"rows" json:"rows"`
		Cols int    `query:"cols" form:"cols" json:"cols"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if req.Rows == 0 || req.Cols == 0 {
		return c.JSON(http.StatusBadRequest, "Rows or Cols can't be zero")
	}

	termMutex.Lock()
	term, exists := activeTerms[req.Id]
	termMutex.Unlock()

	if !exists {
		return c.JSON(http.StatusNotFound, "Terminal not found")
	}

	err := term.SetWindowSize(req.Rows, req.Cols)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, term)
}

func LinkTermDataHandler(c echo.Context) error {
	const TermBufferSize = 8192
	id := c.Param("id")

	termMutex.Lock()
	term, exists := activeTerms[id]
	termMutex.Unlock()

	if !exists {
		return c.JSON(http.StatusNotFound, "Terminal not found")
	}

	websocket.Handler(func(ws *websocket.Conn) {
		defer func() {
			c.Logger().Infof("Destroy term: %s", term.Id)
			termMutex.Lock()
			delete(activeTerms, term.Id)
			termMutex.Unlock()
			term.Close()
			ws.Close()
		}()
		c.Logger().Infof("Linking term: %s", term.Id)
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
			msg := ""
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				c.Logger().Error(err)
				return
			}
			_, err = term.Stdin.Write([]byte(msg))
			if err != nil {
				c.Logger().Error(err)
				return
			}
		}
	}).ServeHTTP(c.Response(), c.Request())

	return nil
}
