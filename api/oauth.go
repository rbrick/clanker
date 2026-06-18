package api

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func (s *Server) handleLinearOAuthCallback(c *echo.Context) error {
	if s.services == nil {
		return writeError(c, http.StatusNotFound, "service manager not configured")
	}
	state, code := c.QueryParam("state"), c.QueryParam("code")
	if state == "" || code == "" {
		return writeError(c, http.StatusBadRequest, "missing state or code")
	}
	conn, err := s.services.CompleteLinearOAuth(c.Request().Context(), state, code)
	if err != nil {
		return writeError(c, http.StatusBadRequest, err.Error())
	}
	return c.HTML(http.StatusOK, "<h1>Linear connected</h1><p>Linear is now connected for "+conn.Platform+" chat. You can close this window and ask Clanker to create Linear tickets.</p>")
}
