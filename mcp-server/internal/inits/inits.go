package inits

import (
	"mcp-server/internal/router"

	"github.com/mszlu521/thunder/config"
	"github.com/mszlu521/thunder/server"
)

func Init(s *server.Server, conf *config.Config) {
	s.RegisterRouters(&router.Event{}, &router.McpRouter{})
}
