package kbs

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/parser/docx"
	"github.com/cloudwego/eino-ext/components/document/parser/pdf"
	"github.com/cloudwego/eino/components/document/parser"
	"github.com/mszlu521/thunder/einos/components/document/parser/epub"
)

func DocxParser(config *docx.Config) (parser.Parser, error) {
	return docx.NewDocxParser(context.Background(), config)
}

func PDFParser(config *pdf.Config) (parser.Parser, error) {
	return pdf.NewPDFParser(context.Background(), config)
}

func HtmlParser(config *HtmlConfig) (parser.Parser, error) {
	return NewHtmlParser(context.Background(), config)
}

func EpubParser(config *epub.Config) (parser.Parser, error) {
	return epub.NewParser(context.Background(), config)
}
