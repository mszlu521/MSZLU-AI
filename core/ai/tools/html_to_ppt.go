package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mszlu521/thunder/ai/einos"
)

// HTMLToPPTTool HTML 转 PPT 工具
type HTMLToPPTTool struct {
	outputDir string
}

type HTMLToPPTConfig struct {
	OutputDir string // PPT 输出目录
}

func NewHTMLToPPTTool(c *HTMLToPPTConfig) einos.InvokeParamTool {
	if c == nil {
		c = &HTMLToPPTConfig{}
	}
	return &HTMLToPPTTool{outputDir: c.OutputDir}
}

func (h *HTMLToPPTTool) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"html_content": {
			Desc:     "HTML 内容字符串",
			Type:     schema.String,
			Required: true,
		},
		"output_path": {
			Desc:     "输出的 PPTX 文件路径（包含文件名和.pptx 扩展名）",
			Type:     schema.String,
			Required: true,
		},
		"title": {
			Desc: "PPT 标题，如果不提供则从 HTML 标题提取",
			Type: schema.String,
		},
		"slides_per_page": {
			Desc: "每页幻灯片的内容数量（默认 5）",
			Type: schema.Integer,
		},
	}
}

func (h *HTMLToPPTTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "html_to_ppt",
		Desc: "将 HTML 内容转换为 PPTX 演示文稿，自动解析 HTML 结构并生成幻灯片（需要安装 Python 和 python-pptx 库）",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"html_content": {
				Desc:     "HTML 内容字符串",
				Type:     schema.String,
				Required: true,
			},
			"output_path": {
				Desc:     "输出的 PPTX 文件路径（包含文件名和.pptx 扩展名）",
				Type:     schema.String,
				Required: true,
			},
			"title": {
				Desc: "PPT 标题，如果不提供则从 HTML 标题提取",
				Type: schema.String,
			},
			"slides_per_page": {
				Desc: "每页幻灯片的内容数量（默认 5）",
				Type: schema.Integer,
			},
		}),
	}, nil
}

// InvokableRun 执行 HTML 到 PPT 的转换
func (h *HTMLToPPTTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var params map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", fmt.Errorf("解析参数失败：%w", err)
	}

	// 获取 HTML 内容
	htmlContent, ok := params["html_content"].(string)
	if !ok || htmlContent == "" {
		return "", fmt.Errorf("html_content 参数为空")
	}

	// 获取输出路径
	outputPath, ok := params["output_path"].(string)
	if !ok || outputPath == "" {
		return "", fmt.Errorf("output_path 参数为空")
	}

	// 确保输出路径有.pptx 扩展名
	if !strings.HasSuffix(outputPath, ".pptx") {
		outputPath = outputPath + ".pptx"
	}

	// 如果配置了 outputDir，构建完整路径
	if h.outputDir != "" && !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(h.outputDir, outputPath)
	}

	// 使用 Python 脚本创建 PPTX 文件
	err := createPPTXWithPython(outputPath, htmlContent)
	if err != nil {
		return "", fmt.Errorf("创建 PPTX 失败：%w", err)
	}

	// 返回成功信息
	result := map[string]string{
		"status":      "success",
		"message":     fmt.Sprintf("PPT 生成成功：%s", outputPath),
		"output_path": outputPath,
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// createPPTXWithPython 使用 Python 脚本生成 PPTX 文件
// 使用 BeautifulSoup 解析 HTML，支持 Tailwind CSS 样式映射
func createPPTXWithPython(outputPath string, htmlContent string) error {
	// 创建输出目录（如果不存在）
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败：%w", err)
	}

	// 检查 Python 是否可用
	pythonCmd := exec.Command("python", "--version")
	if err := pythonCmd.Run(); err != nil {
		// 尝试 python3
		pythonCmd = exec.Command("python3", "--version")
		if err := pythonCmd.Run(); err != nil {
			return fmt.Errorf("Python 未安装或不可用。请先安装 Python，然后执行：pip install python-pptx beautifulsoup4")
		}
	}

	// 创建临时 HTML 文件
	tempHTML := filepath.Join(dir, "temp_html_"+fmt.Sprintf("%d", time.Now().UnixNano())+".html")
	fullHTML := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Temp Presentation</title>
</head>
<body class="bg-black text-white overflow-hidden font-sans">
<div id="presentation" class="relative w-full h-screen glow-bg">
` + htmlContent + `
</div>
</body>
</html>`

	if err := os.WriteFile(tempHTML, []byte(fullHTML), 0644); err != nil {
		return fmt.Errorf("写入临时HTML文件失败：%w", err)
	}
	defer os.Remove(tempHTML) // 清理临时HTML文件

	// 创建临时 Python 脚本
	tempScript := filepath.Join(dir, "generate_ppt_bs_"+fmt.Sprintf("%d", time.Now().UnixNano())+".py")
	pythonCode := `#!/usr/bin/env python3
"""HTML 转 PPT - 使用 BeautifulSoup 解析 Tailwind 样式"""

import re
import sys

from bs4 import BeautifulSoup, NavigableString
from pptx import Presentation
from pptx.dml.color import RGBColor
from pptx.enum.shapes import MSO_SHAPE
from pptx.enum.text import PP_ALIGN
from pptx.util import Inches, Pt

# Tailwind 字体大小映射
FONT_SIZE_MAP = {
    "text-7xl": Pt(70),
    "text-6xl": Pt(60),
    "text-5xl": Pt(48),
    "text-4xl": Pt(36),
    "text-3xl": Pt(30),
    "text-2xl": Pt(24),
    "text-xl": Pt(20),
    "text-lg": Pt(18),
}

# 颜色映射
COLOR_MAP = {
    "text-white": RGBColor(255, 255, 255),
    "text-gray-300": RGBColor(209, 213, 219),
    "text-gray-400": RGBColor(156, 163, 175),
    "text-gray-500": RGBColor(107, 114, 128),
}


def get_font_size(class_list):
    for cls in class_list:
        if cls in FONT_SIZE_MAP:
            return FONT_SIZE_MAP[cls]
    return Pt(24)


def get_color(class_list):
    for cls in class_list:
        if cls in COLOR_MAP:
            return COLOR_MAP[cls]
    return RGBColor(255, 255, 255)


def is_bold(class_list):
    return "font-bold" in class_list or "font-black" in class_list


def is_grid_layout(class_list):
    """判断是否是 grid 两列布局"""
    return "grid-cols-2" in class_list or "md:grid-cols-2" in class_list


def is_flex_row(class_list):
    """判断是否是水平 flex 布局"""
    return "flex-row" in class_list or ("flex" in class_list and "flex-col" not in class_list)


def clean_text(text):
    """清理文本"""
    if not text:
        return ""
    text = text.replace("&nbsp;", " ")
    text = " ".join(text.split())  # 合并多个空格
    return text.strip()


def extract_slide_content(slide_html):
    """提取 slide 中的所有内容元素"""
    soup = BeautifulSoup(slide_html, "html.parser")
    slide_div = soup.find("div", class_=re.compile("slide"))

    if not slide_div:
        return [], False

    elements = []
    seen_texts = set()
    has_two_column = False

    def process_element(elem, depth=0, in_grid=False):
        """递归处理元素"""
        nonlocal has_two_column

        if elem is None or isinstance(elem, NavigableString):
            return

        if elem.name in ["script", "style"]:
            return

        class_list = elem.get("class", [])

        # 检查是否是 grid 两列布局
        if elem.name == "div" and is_grid_layout(class_list):
            has_two_column = True
            # 提取两列内容
            cols = elem.find_all("div", recursive=False)
            left_items = []
            right_items = []

            for i, col in enumerate(cols[:2]):  # 只取前两列
                col_elements = []
                for child in col.find_all(["p", "h1", "h2", "h3", "div"], recursive=True):
                    text = clean_text(child.get_text())
                    if text and text not in seen_texts:
                        child_classes = child.get("class", [])
                        col_elements.append(
                            {
                                "text": text,
                                "classes": child_classes,
                                "is_horizontal": False,
                                "is_list": False,
                            }
                        )
                        seen_texts.add(text)

                if i == 0:
                    left_items = col_elements
                else:
                    right_items = col_elements

            if left_items or right_items:
                elements.append({"type": "two_column", "left": left_items, "right": right_items})
            return

        # 检查是否是水平 flex 容器 (流程箭头行)
        if elem.name == "div" and is_flex_row(class_list) and not in_grid:
            parts = []
            for child in elem.descendants:
                if isinstance(child, NavigableString):
                    text = clean_text(str(child))
                    if text:
                        parts.append(text)
                elif child.name in ["span", "p"]:
                    text = clean_text(child.get_text())
                    if text:
                        parts.append(text)

            full_text = " ".join(parts)
            if full_text and full_text not in seen_texts and len(full_text) > 1:
                elements.append(
                    {
                        "type": "text",
                        "text": full_text,
                        "classes": class_list,
                        "is_horizontal": True,
                        "is_list": False,
                    }
                )
                seen_texts.add(full_text)
            return

        # 如果是文本元素
        if elem.name in ["h1", "h2", "h3", "p"]:
            text = clean_text(elem.get_text())
            if text and text not in seen_texts:
                is_list = text.startswith("•") or elem.find_parent("li") is not None
                elements.append(
                    {
                        "type": "text",
                        "text": text,
                        "classes": class_list,
                        "is_horizontal": False,
                        "is_list": is_list,
                    }
                )
                seen_texts.add(text)
                return

        # 递归处理子元素
        for child in elem.children:
            process_element(child, depth + 1, in_grid or is_grid_layout(class_list))

    process_element(slide_div)
    return elements, has_two_column


def create_slide(prs, elements, is_first=False):
    """创建单个 slide"""
    slide_layout = prs.slide_layouts[6]
    slide = prs.slides.add_slide(slide_layout)

    # 黑色背景
    bg = slide.shapes.add_shape(MSO_SHAPE.RECTANGLE, 0, 0, prs.slide_width, prs.slide_height)
    bg.fill.solid()
    bg.fill.fore_color.rgb = RGBColor(10, 10, 10)
    bg.line.fill.background()

    if not elements:
        return slide

    # 计算起始位置（垂直居中）
    total_items = len([e for e in elements if e.get("type") == "text"])
    total_height = total_items * 0.5 + sum(1.0 for e in elements if e.get("type") == "two_column")
    y_pos = Inches((7.5 - total_height) / 2)
    if y_pos < Inches(0.8):
        y_pos = Inches(0.8)

    for elem_data in elements:
        elem_type = elem_data.get("type", "text")

        if elem_type == "two_column":
            # 处理两列布局
            left_items = elem_data.get("left", [])
            right_items = elem_data.get("right", [])

            col_y = y_pos
            max_height = Inches(0)

            # 左列
            for item in left_items:
                txBox = slide.shapes.add_textbox(Inches(0.8), col_y, Inches(5.5), Inches(1.0))
                tf = txBox.text_frame
                tf.word_wrap = True
                p = tf.paragraphs[0]
                p.text = item["text"]
                p.alignment = PP_ALIGN.CENTER
                p.font.size = get_font_size(item["classes"])
                p.font.color.rgb = get_color(item["classes"])
                p.font.bold = is_bold(item["classes"])
                col_y += Inches(0.6)

            if left_items:
                max_height = max(max_height, col_y - y_pos)

            # 右列
            col_y = y_pos
            for item in right_items:
                txBox = slide.shapes.add_textbox(Inches(7.0), col_y, Inches(5.5), Inches(1.0))
                tf = txBox.text_frame
                tf.word_wrap = True
                p = tf.paragraphs[0]
                p.text = item["text"]
                p.alignment = PP_ALIGN.CENTER
                p.font.size = get_font_size(item["classes"])
                p.font.color.rgb = get_color(item["classes"])
                p.font.bold = is_bold(item["classes"])
                col_y += Inches(0.6)

            if right_items:
                max_height = max(max_height, col_y - y_pos)

            y_pos += max_height + Inches(0.3)

        else:
            # 普通文本元素
            text = elem_data.get("text", "")
            classes = elem_data.get("classes", [])
            is_list = elem_data.get("is_list", False)

            if not text:
                continue

            # 列表项使用更小的间距
            box_height = Inches(0.5) if is_list else Inches(0.8)

            txBox = slide.shapes.add_textbox(Inches(0.8), y_pos, Inches(11.733), box_height)
            tf = txBox.text_frame
            tf.word_wrap = True
            p = tf.paragraphs[0]

            if is_list and not text.startswith("•"):
                p.text = "• " + text
            else:
                p.text = text

            p.alignment = PP_ALIGN.CENTER
            p.font.size = get_font_size(classes)
            p.font.color.rgb = get_color(classes)
            p.font.bold = is_bold(classes)

            # 列表项间距更小 (0.5)，普通文本保持 (0.8)
            y_pos += Inches(0.5) if is_list else Inches(0.8)

    return slide


def main():
    if len(sys.argv) < 3:
        print("Usage: python script.py <input.html> <output.pptx>")
        sys.exit(1)

    input_file = sys.argv[1]
    output_file = sys.argv[2]

    with open(input_file, "r", encoding="utf-8") as f:
        html_content = f.read()

    # 提取所有 slide
    soup = BeautifulSoup(html_content, "html.parser")
    slides = soup.find_all("div", class_=re.compile("slide"))

    print(f"Found {len(slides)} slides")

    prs = Presentation()
    prs.slide_width = Inches(13.333)
    prs.slide_height = Inches(7.5)

    for i, slide_div in enumerate(slides):
        print(f"Processing slide {i + 1}...")
        elements, has_two_col = extract_slide_content(str(slide_div))
        print(f"  - Found {len(elements)} elements, two_column={has_two_col}")
        create_slide(prs, elements, is_first=(i == 0))

    prs.save(output_file)
    print(f"PPT saved: {output_file}")


if __name__ == "__main__":
    main()
`

	// 写入临时脚本文件
	if err := os.WriteFile(tempScript, []byte(pythonCode), 0755); err != nil {
		return fmt.Errorf("写入临时脚本文件失败：%w", err)
	}
	defer os.Remove(tempScript) // 清理临时脚本文件

	// 执行 Python 脚本
	cmd := exec.Command("python", tempScript, tempHTML, outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// 尝试 python3
		cmd = exec.Command("python3", tempScript, tempHTML, outputPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("执行 Python 脚本失败：%w", err)
		}
	}

	return nil
}
