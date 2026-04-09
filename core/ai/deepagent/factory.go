package deepagent

import (
	"context"
	"fmt"
	"model"
)

type Factory struct {
}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Create(ctx context.Context, cfg *UniversalDeepAgentConfig) (*UniversalDeepAgent, error) {
	if cfg.Agent == nil {
		return nil, fmt.Errorf("agent is nil")
	}
	if cfg.Agent.Mode != model.DeepAgentMode {
		return nil, fmt.Errorf("agent mode is not deep")
	}
	if cfg.Agent.DeepConfig == nil {
		return nil, fmt.Errorf("deep config is nil")
	}
	deepConfig := cfg.Agent.DeepConfig.ToDeepAgentConfig()
	return NewUniversalDeepAgent(ctx, cfg, deepConfig)
}
