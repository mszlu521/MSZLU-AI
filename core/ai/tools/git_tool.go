package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mszlu521/thunder/ai/einos"
)

// GitTool 定义 Git 工具
type GitTool struct {
	Name        string
	Description string
}

func (g *GitTool) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"command": {
			Type:     schema.String,
			Desc:     "Git 命令，如 status, add, commit, diff, log, checkout 等",
			Required: true,
		},
		"args": {
			Type:     schema.String,
			Desc:     "命令参数，如 -A, -m 'message', --cached 等",
			Required: false,
		},
		"working_dir": {
			Type:     schema.String,
			Desc:     "工作目录，指定git命令执行的目录路径，默认为当前目录",
			Required: false,
		},
	}
}

// GitToolArgs 定义 Git 工具的参数结构
type GitToolArgs struct {
	Command    string `json:"command"`     // Git 命令，如 "status", "add", "commit", "diff" 等
	Args       string `json:"args"`        // 命令参数，如 "-A" 或 "-m 'commit message'"
	WorkingDir string `json:"working_dir"` // 工作目录
}

// Info 返回工具信息
func (g *GitTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: g.Name,
		Desc: g.Description,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "Git 命令，如 status, add, commit, diff, log, checkout 等",
				Required: true,
			},
			"args": {
				Type:     schema.String,
				Desc:     "命令参数，如 -A, -m 'message', --cached 等",
				Required: false,
			},
			"working_dir": {
				Type:     schema.String,
				Desc:     "工作目录，指定git命令执行的目录路径，默认为当前目录",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行 Git 命令
func (g *GitTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args GitToolArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	// 构建完整的 Git 命令
	cmdParts := []string{"git", args.Command}
	if args.Args != "" {
		cmdParts = append(cmdParts, strings.Fields(args.Args)...)
	}

	// 执行 Git 命令
	var cmd *exec.Cmd
	if args.WorkingDir != "" {
		cmd = exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
		cmd.Dir = args.WorkingDir
	} else {
		// 如果没有指定工作目录，则尝试查找最近的git仓库目录
		gitDir, err := findGitRoot()
		if err != nil {
			// 如果没找到git仓库，使用当前目录
			cmd = exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
		} else {
			cmd = exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
			cmd.Dir = gitDir
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("执行 Git 命令失败: %s\n错误输出: %s", strings.Join(cmdParts, " "), string(output)), nil
	}

	return string(output), nil
}

// findGitRoot 查找最近的git仓库根目录
func findGitRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// 从当前目录开始向上遍历目录树直到找到.git目录
	dir := currentDir
	for {
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir, nil
		}

		// 检查当前目录是否是git仓库的根目录
		cmd := exec.Command("git", "rev-parse", "--git-dir")
		cmd.Dir = dir
		if output, err := cmd.Output(); err == nil {
			// 输出包含.git目录，说明这是git仓库的根目录
			gitDir := strings.TrimSpace(string(output))
			if strings.HasSuffix(gitDir, ".git") || strings.Contains(gitDir, "/.git") || strings.Contains(gitDir, "\\.git") {
				return dir, nil
			}
		}

		// 检查是否到达文件系统的根目录
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// 已经到达根目录
			break
		}
		dir = parentDir
	}

	return "", fmt.Errorf("未找到git仓库")
}

// NewGitTool 创建一个新的 Git 工具实例
func NewGitTool() einos.InvokeParamTool {
	return &GitTool{
		Name:        "git_command",
		Description: "执行 Git 命令的工具，支持所有 Git 子命令及其参数。例如：git status, git add, git commit, git diff 等。",
	}
}
