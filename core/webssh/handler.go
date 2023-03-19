package webssh

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

func Index(c echo.Context) error {
	return c.Render(http.StatusOK, "term.template", termStore.All())
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

func CreateTermHandler(c echo.Context) error {
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

func SetTermWindowSizeHandler(c echo.Context) error {
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

func LinkTermDataHandler(c echo.Context) error {
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
