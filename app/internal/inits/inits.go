package inits

import (
	"app/internal/router"

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
	s.RegisterRouters(
		&router.Event{},
		&router.AuthRouter{},
		&router.SubscriptionRouter{},
	)
}
