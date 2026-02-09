package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mszlu521/thunder/ai/einos"
)

// GitCommitTool 专门用于处理 Git 提交流程的工具
type GitCommitTool struct {
	Name        string
	Description string
}

func (g *GitCommitTool) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"action": {
			Type:     schema.String,
			Desc:     "操作类型: status(获取状态), diff(查看差异), add(添加文件), commit(提交), validate(验证提交信息)",
			Required: true,
		},
		"message": {
			Type:     schema.String,
			Desc:     "提交信息（仅在 commit 操作时使用）",
			Required: false,
		},
		"files": {
			Type:     schema.String,
			Desc:     "指定文件路径（仅在 add 操作时使用）",
			Required: false,
		},
		"working_dir": {
			Type:     schema.String,
			Desc:     "工作目录，指定git命令执行的目录路径，默认为当前目录",
			Required: false,
		},
	}
}

// GitCommitToolArgs 定义 Git 提交工具的参数结构
type GitCommitToolArgs struct {
	Action     string `json:"action"`      // 操作类型: status, diff, add, commit, validate
	Message    string `json:"message"`     // 提交信息
	Files      string `json:"files"`       // 指定文件
	WorkingDir string `json:"working_dir"` // 工作目录
}

// Info 返回工具信息
func (g *GitCommitTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: g.Name,
		Desc: g.Description,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "操作类型: status(获取状态), diff(查看差异), add(添加文件), commit(提交), validate(验证提交信息)",
				Required: true,
			},
			"message": {
				Type:     schema.String,
				Desc:     "提交信息（仅在 commit 操作时使用）",
				Required: false,
			},
			"files": {
				Type:     schema.String,
				Desc:     "指定文件路径（仅在 add 操作时使用）",
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

// InvokableRun 执行 Git 提交相关操作
func (g *GitCommitTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args GitCommitToolArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	switch args.Action {
	case "status":
		return g.gitStatus(ctx, args.WorkingDir)
	case "diff":
		return g.gitDiff(ctx, args.WorkingDir)
	case "add":
		return g.gitAdd(ctx, args.Files, args.WorkingDir)
	case "commit":
		return g.gitCommit(ctx, args.Message, args.WorkingDir)
	case "validate":
		return g.validateCommitMessage(ctx, args.Message)
	default:
		return "", fmt.Errorf("不支持的操作: %s", args.Action)
	}
}

// gitStatus 获取 Git 状态
func (g *GitCommitTool) gitStatus(ctx context.Context, workingDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--short")
	if workingDir != "" {
		cmd.Dir = workingDir
	} else {
		gitDir, err := findGitRoot()
		if err != nil {
			// 如果没找到git仓库，使用当前目录
			wd, _ := os.Getwd()
			cmd.Dir = wd
		} else {
			cmd.Dir = gitDir
		}
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("获取 Git 状态失败: %s", string(output)), nil
	}
	if string(output) == "" {
		return "工作区干净，没有未提交的更改", nil
	}
	return string(output), nil
}

// gitDiff 获取差异
func (g *GitCommitTool) gitDiff(ctx context.Context, workingDir string) (string, error) {
	cmd1 := exec.CommandContext(ctx, "git", "diff", "--cached")
	if workingDir != "" {
		cmd1.Dir = workingDir
	} else {
		gitDir, err := findGitRoot()
		if err != nil {
			// 如果没找到git仓库，使用当前目录
			wd, _ := os.Getwd()
			cmd1.Dir = wd
		} else {
			cmd1.Dir = gitDir
		}
	}
	cachedOutput, _ := cmd1.CombinedOutput()

	cmd2 := exec.CommandContext(ctx, "git", "diff")
	if workingDir != "" {
		cmd2.Dir = workingDir
	} else {
		gitDir, err := findGitRoot()
		if err != nil {
			// 如果没找到git仓库，使用当前目录
			wd, _ := os.Getwd()
			cmd2.Dir = wd
		} else {
			cmd2.Dir = gitDir
		}
	}
	uncachedOutput, _ := cmd2.CombinedOutput()

	result := ""
	if string(cachedOutput) != "" {
		result += "=== 已暂存的更改 ===\n" + string(cachedOutput) + "\n"
	}
	if string(uncachedOutput) != "" {
		result += "=== 未暂存的更改 ===\n" + string(uncachedOutput)
	}
	if result == "" {
		result = "没有差异"
	}
	return result, nil
}

// gitAdd 添加文件到暂存区
func (g *GitCommitTool) gitAdd(ctx context.Context, files string, workingDir string) (string, error) {
	cmdArgs := []string{"add"}
	if files == "" || strings.TrimSpace(files) == "-A" {
		cmdArgs = append(cmdArgs, "-A")
	} else {
		cmdArgs = append(cmdArgs, strings.Fields(files)...)
	}

	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	if workingDir != "" {
		cmd.Dir = workingDir
	} else {
		gitDir, err := findGitRoot()
		if err != nil {
			// 如果没找到git仓库，使用当前目录
			wd, _ := os.Getwd()
			cmd.Dir = wd
		} else {
			cmd.Dir = gitDir
		}
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("添加文件失败: %s", string(output)), nil
	}
	return fmt.Sprintf("成功添加文件: %s", strings.Join(cmdArgs[1:], " ")), nil
}

// gitCommit 执行提交
func (g *GitCommitTool) gitCommit(ctx context.Context, message string, workingDir string) (string, error) {
	if message == "" {
		return "提交信息不能为空", nil
	}

	cmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	if workingDir != "" {
		cmd.Dir = workingDir
	} else {
		gitDir, err := findGitRoot()
		if err != nil {
			// 如果没找到git仓库，使用当前目录
			wd, _ := os.Getwd()
			cmd.Dir = wd
		} else {
			cmd.Dir = gitDir
		}
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("提交失败: %s", string(output)), nil
	}
	return fmt.Sprintf("提交成功:\n%s", string(output)), nil
}

// validateCommitMessage 验证提交信息
func (g *GitCommitTool) validateCommitMessage(ctx context.Context, message string) (string, error) {
	if message == "" {
		return "提交信息不能为空", nil
	}

	// 这里可以调用你现有的 validate_message.py 脚本来验证
	cmd := exec.CommandContext(ctx, "python", "skills/git-commit/scripts/validate_message.py", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 如果验证失败，返回验证错误信息
		return fmt.Sprintf("提交信息验证结果:\n%s", string(output)), nil
	}
	return fmt.Sprintf("提交信息验证通过\n%s", string(output)), nil
}

//// findGitRoot 查找最近的git仓库根目录
//func findGitRoot() (string, error) {
//	currentDir, err := os.Getwd()
//	if err != nil {
//		return "", err
//	}
//
//	// 从当前目录开始向上遍历目录树直到找到.git目录
//	dir := currentDir
//	for {
//		gitPath := filepath.Join(dir, ".git")
//		if _, err := os.Stat(gitPath); err == nil {
//			return dir, nil
//		}
//
//		// 检查当前目录是否是git仓库的根目录
//		cmd := exec.Command("git", "rev-parse", "--git-dir")
//		cmd.Dir = dir
//		if output, err := cmd.Output(); err == nil {
//			// 输出包含.git目录，说明这是git仓库的根目录
//			gitDir := strings.TrimSpace(string(output))
//			if strings.HasSuffix(gitDir, ".git") || strings.Contains(gitDir, "/.git") || strings.Contains(gitDir, "\\.git") {
//				return dir, nil
//			}
//		}
//
//		// 检查是否到达文件系统的根目录
//		parentDir := filepath.Dir(dir)
//		if parentDir == dir {
//			// 已经到达根目录
//			break
//		}
//		dir = parentDir
//	}
//
//	return "", fmt.Errorf("未找到git仓库")
//}

// NewGitCommitTool 创建一个新的 Git 提交工具实例
func NewGitCommitTool() einos.InvokeParamTool {
	return &GitCommitTool{
		Name:        "git_commit_workflow",
		Description: "Git 提交流程专用工具，支持状态检查、差异查看、文件添加、提交和提交信息验证等操作。",
	}
}
