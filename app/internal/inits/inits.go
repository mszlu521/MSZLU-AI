package inits

import (
	"app/internal/router"
	"core/ai"
	"core/ai/tools"

	"github.com/mszlu521/thunder/config"
	"github.com/mszlu521/thunder/database"
	"github.com/mszlu521/thunder/logs"
	"github.com/mszlu521/thunder/server"
	"github.com/mszlu521/thunder/tools/jwt"
)

func Init(s *server.Server, conf *config.Config) {
	//初始化数据库
	database.InitPostgres(conf.DB.Postgres)
	logs.Infof("数据库初始化完成")
	//初始化redis
	database.InitRedis(conf.DB.Redis)
	logs.Infof("redis初始化完成")
	//初始化jwt
	jwt.Init(conf.Jwt.GetSecret())
	logs.Infof("jwt初始化完成")
	//注册系统工具
	registerTools()
	logs.Infof("系统工具初始化完成")
	//初始化工作流执行器
	ai.Init()
	logs.Infof("工作流执行器初始化完成")
	closeFuncs := s.RegisterRouters(
		&router.Event{},
		&router.HealthRouter{},
		&router.AuthRouter{},
		&router.SubscriptionRouter{},
		&router.AgentRouter{},
		&router.LLMRouter{},
		&router.ToolRouter{},
		&router.KnowledgeBaseRouter{},
		&router.A2ARouter{},
		&router.WorkflowRouter{},
		&router.NodeRouter{},
		&router.SkillRouter{},
	)
	// 注册事件处理器
	eventRouter := &router.Event{}
	eventRouter.Register()
	logs.Infof("路由初始化完成")
	s.Close = func() {
		for _, f := range closeFuncs {
			err := f()
			if err != nil {
				logs.Error("close func error", "error", err)
				return
			}
		}
	}
}

func registerTools() {
	err := tools.InitK8sClient()
	if err != nil {
		logs.Error("init k8s client error", "error", err)
	}
	tools.RegisterSystemTools(
		tools.NewWeatherTool(&tools.WeatherConfig{ApiKey: tools.ApiKey}),
		tools.NewGitTool(), tools.NewGitCommitTool(),
		tools.NewK8sResourceQueryTool(),
		tools.NewK8sLogsTool(),
		tools.NewK8sResourceActionTool(),
		tools.NewK8sHealthCheckTool(),
		tools.NewFileWriteTool(&tools.FileWriteConfig{}),
		tools.NewHTMLToPPTTool(&tools.HTMLToPPTConfig{}),
	)
}
