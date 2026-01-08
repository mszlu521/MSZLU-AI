package inits

import (
	"app/internal/router"
	"core/ai/tools"

	"github.com/mszlu521/thunder/config"
	"github.com/mszlu521/thunder/database"
	"github.com/mszlu521/thunder/server"
	"github.com/mszlu521/thunder/tools/jwt"
)

func Init(s *server.Server, conf *config.Config) {
	//初始化数据库
	database.InitPostgres(conf.DB.Postgres)
	//初始化redis
	database.InitRedis(conf.DB.Redis)
	//初始化jwt
	jwt.Init(conf.Jwt.GetSecret())
	//注册系统工具
	registerTools()
	s.RegisterRouters(
		&router.Event{},
		&router.AuthRouter{},
		&router.SubscriptionRouter{},
		&router.AgentRouter{},
		&router.LLMRouter{},
		&router.ToolRouter{},
		&router.KnowledgeBaseRouter{},
	)
}

func registerTools() {
	tools.RegisterSystemTools(
		tools.NewWeatherTool(&tools.WeatherConfig{ApiKey: tools.ApiKey}),
	)
}
