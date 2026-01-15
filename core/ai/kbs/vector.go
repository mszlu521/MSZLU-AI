package kbs

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

type SearchFilter map[string]any

type VectorStore interface {
	Store(ctx context.Context, docs []*schema.Document) error
	Search(ctx context.Context, query string, topK int, filters SearchFilter) ([]*schema.Document, error)
}
