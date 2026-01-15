package kbs

import "strings"

type FileType string

const (
	Markdown FileType = "markdown"
	Text     FileType = "text"
	PDF      FileType = "pdf"
	Excel    FileType = "excel"
	Docx     FileType = "docx"
	Html     FileType = "html"
	Epub     FileType = "epub"
	Unknown  FileType = "unknown"
)

func FromExtension(ext string) FileType {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	switch ext {
	case "md", "markdown":
		return Markdown
	case "txt":
		return Text
	case "pdf":
		return PDF
	case "xlsx", "xls":
		return Excel
	case "docx", "doc":
		return Docx
	case "html", "htm":
		return Html
	case "epub":
		return Epub
	default:
		return Text
	}
}
