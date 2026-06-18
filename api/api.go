package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/rbrick/clanker/snippets"
)

type Server struct {
	addr     string
	snippets *snippets.Snippets
}

func NewServer(addr string, snippets *snippets.Snippets) *Server {
	return &Server{addr: addr, snippets: snippets}
}

func (s *Server) Start(ctx context.Context) error {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderContentType},
	}))

	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/snippet/:id", s.handleSnippet)

	srv := &http.Server{Addr: s.addr, Handler: e}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	return srv.ListenAndServe()
}

func (s *Server) handleSnippet(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return writeError(c, http.StatusBadRequest, "invalid UUID")
	}
	snippet, err := s.snippets.GetSnippetByID(id)
	if err != nil {
		return writeError(c, http.StatusInternalServerError, err.Error())
	}
	if snippet == nil {
		return writeError(c, http.StatusNotFound, "snippet not found")
	}
	return c.JSON(http.StatusOK, snippet)
}

func writeError(c *echo.Context, status int, msg string) error {
	return c.JSON(status, map[string]string{"error": msg})
}
