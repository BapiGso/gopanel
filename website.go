package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func Website(c echo.Context) error {
	return c.String(http.StatusOK, "123")
}
