package kbs

import (
	"context"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"
)

const (
	MetaKeyTitle   = "_title"
	MetaKeyDesc    = "_description"
	MetaKeyLang    = "_language"
	MetaKeyCharset = "_charset"
	MetaKeySource  = "_source"
)

var _ parser.Parser = (*Parser)(nil)

type HtmlConfig struct {
	// content selector of goquery. eg: body for <body>, #id for <div id="id">
	Selector *string
}

var (
	BodySelector = "body"
)

func NewHtmlParser(ctx context.Context, conf *HtmlConfig) (*Parser, error) {
	if conf == nil {
		conf = &HtmlConfig{}
	}

	return &Parser{
		conf: conf,
	}, nil
}

type Parser struct {
	conf *HtmlConfig
}

func (p *Parser) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) ([]*schema.Document, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	option := parser.GetCommonOptions(&parser.Options{}, opts...)

	// 1. 获取元数据
	meta, err := p.getMetaData(ctx, doc)
	if err != nil {
		return nil, err
	}
	meta[MetaKeySource] = option.URI

	if option.ExtraMeta != nil {
		for k, v := range option.ExtraMeta {
			meta[k] = v
		}
	}

	// 2. 核心改动：获取 HTML 结构而非纯文本
	var htmlContent string
	selector := BodySelector
	if p.conf.Selector != nil {
		selector = *p.conf.Selector
	}

	// 获取指定选择器内的 InnerHTML
	// 如果你的 HTML 是 MD 生成的，通常我们关注 <body> 或某个特定的容器
	htmlContent, err = doc.Find(selector).Html()
	if err != nil {
		return nil, err
	}

	// 3. 使用 bluemonday 保留语义化标签
	// UGCPolicy 会保留: h1, h2, h3, h4, h5, h6, p, b, i, strong, em, a, ul, ol, li, table, thead, tbody, tr, th, td
	// 它会剔除: script, style, object, iframe, 以及所有 on* 事件属性
	policy := bluemonday.UGCPolicy()
	sanitized := policy.Sanitize(htmlContent)

	document := &schema.Document{
		Content:  strings.TrimSpace(sanitized),
		MetaData: meta,
	}

	return []*schema.Document{
		document,
	}, nil
}

// getMetaData 保持不变，用于提取标题、描述等
func (p *Parser) getMetaData(ctx context.Context, doc *goquery.Document) (map[string]any, error) {
	meta := map[string]any{}

	title := doc.Find("title")
	if title != nil {
		if t := title.Text(); t != "" {
			meta[MetaKeyTitle] = t
		}
	}

	description := doc.Find("meta[name=description]")
	if description != nil {
		if desc := description.AttrOr("content", ""); desc != "" {
			meta[MetaKeyDesc] = desc
		}
	}

	htmlNode := doc.Find("html")
	if htmlNode != nil {
		if language := htmlNode.AttrOr("lang", ""); language != "" {
			meta[MetaKeyLang] = language
		}
	}

	charset := doc.Find("meta[charset]")
	if charset != nil {
		if c := charset.AttrOr("charset", ""); c != "" {
			meta[MetaKeyCharset] = c
		}
	}

	return meta, nil
}
