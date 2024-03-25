package term

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
	"io"
	"net/http"
)

func Index(c echo.Context) error {
	return c.Render(http.StatusOK, "term.template", termStore.All())
}

func CreateTermHandler(c echo.Context) error {
	req := &struct {
		Host     string `query:"host" form:"host" json:"host"`
		Port     int    `query:"port" form:"port" json:"port"`
		Username string `query:"user" form:"user" json:"user"`
		Password string `query:"pwd"  form:"pwd" json:"pwd"`
		Rows     int    `query:"rows" form:"rows" json:"rows"`
		Cols     int    `query:"cols" form:"cols" json:"cols"`
	}{}

	if err := c.Bind(req); err != nil {
		return err
	}

	if req.Host == "" || req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, "host or user or password not provided")
	}

	term, err := termStore.New(TermOption{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return err
	}
	c.SetCookie(&http.Cookie{
		Name:  term.Name(),
		Value: term.Id,
	})

	return c.NoContent(200)
}

func SetTermWindowSizeHandler(c echo.Context) error {
	req := &struct {
		Rows int `query:"rows" form:"rows" json:"rows"`
		Cols int `query:"cols" form:"cols" json:"cols"`
	}{}

	if err := c.Bind(req); err != nil {
		return err
	}

	if req.Rows == 0 || req.Cols == 0 {
		return c.JSON(http.StatusBadRequest, "Rows or cols can't be zero")
	}

	term, err := termStore.Get(c.Param("id"))
	if err != nil {
		return err
	}
	defer termStore.Put(term.Id)

	err = term.SetWindowSize(req.Rows, req.Cols)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, term)
}

func LinkTermDataHandler(c echo.Context) error {
	const TermBufferSize = 8192
	term, err := termStore.Lookup(c.Param("id"))
	if err != nil {
		return err
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
