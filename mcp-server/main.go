package main

import (
	"mcp-server/internal/inits"

	"github.com/mszlu521/thunder/config"
	"github.com/mszlu521/thunder/logs"
	"github.com/mszlu521/thunder/server"
)

func main() {
	//加载etc/config.yml中的配置
	config.Init()
	conf := config.GetConfig()
	//初始化日志
	logs.Init(conf.Log)
	//初始化Gin服务
	s := server.NewServer(conf)
	//初始化各个模块
	inits.Init(s, conf)
	//启动服务
	s.Start()
}
