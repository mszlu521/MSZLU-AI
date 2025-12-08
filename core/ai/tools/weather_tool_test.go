package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewWeatherTool(t *testing.T) {
	weatherTool := NewWeatherTool(&WeatherConfig{
		ApiKey: ApiKey,
	})
	params := map[string]string{
		"city":       "北京",
		"extensions": "all",
	}
	marshal, _ := json.Marshal(params)
	invokableRun, err := weatherTool.InvokableRun(context.Background(), string(marshal))
	if err != nil {
		t.Errorf("InvokableRun() error = %v", err)
	}
	t.Logf("InvokableRun() = %v", invokableRun)

}
