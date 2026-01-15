package kbs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/indexer/es8"
	reEs8 "github.com/cloudwego/eino-ext/components/retriever/es8"
	"github.com/cloudwego/eino-ext/components/retriever/es8/search_mode"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/schema"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

type ESVectorStore struct {
	indexer   *es8.Indexer
	retriever *reEs8.Retriever
}

func NewESVectorStore(
	ctx context.Context,
	esClient *elasticsearch.Client,
	index string,
	embedder embedding.Embedder) (*ESVectorStore, error) {
	indexer, err := es8.NewIndexer(ctx, &es8.IndexerConfig{
		Client: esClient,
		Index:  index,
		DocumentToFields: func(ctx context.Context, d *schema.Document) (field2Value map[string]es8.FieldValue, err error) {
			return map[string]es8.FieldValue{
				"content": {
					Value:    d.Content,
					EmbedKey: "content_vector",
				},
				"doc_id": {
					Value: d.MetaData["doc_id"],
				},
				"parent_id": {
					Value: d.MetaData["parent_id"],
				},
				"metadata": {
					Value: d.MetaData,
				},
			}, nil
		},
		Embedding: embedder,
	})
	if err != nil {
		return nil, err
	}
	retriever, err := reEs8.NewRetriever(ctx, &reEs8.RetrieverConfig{
		Client: esClient,
		Index:  index,
		SearchMode: search_mode.SearchModeApproximate(&search_mode.ApproximateConfig{
			VectorFieldName: "content_vector",
		}),
		ResultParser: func(ctx context.Context, hit types.Hit) (doc *schema.Document, err error) {
			doc = &schema.Document{
				ID:       *hit.Id_,
				Content:  "",
				MetaData: map[string]interface{}{},
			}
			var src map[string]any
			err = json.Unmarshal(hit.Source_, &src)
			if err != nil {
				return nil, err
			}
			doc.Content = src["content"].(string)
			doc.MetaData = src["metadata"].(map[string]any)
			if hit.Score_ != nil {
				doc.WithScore(float64(*hit.Score_))
			}
			return doc, nil
		},
		Embedding: embedder,
	})
	if err != nil {
		return nil, err
	}
	return &ESVectorStore{
		indexer:   indexer,
		retriever: retriever,
	}, nil
}

func (s *ESVectorStore) Store(ctx context.Context, docs []*schema.Document) error {
	_, err := s.indexer.Store(ctx, docs)
	return err
}
func (s *ESVectorStore) Search(ctx context.Context, query string, topK int, filters SearchFilter) ([]*schema.Document, error) {
	var esFilters []types.Query
	for k, v := range filters {
		esFilters = append(esFilters, types.Query{
			Term: map[string]types.TermQuery{
				fmt.Sprintf("metadata.%s", k): {Value: v},
			},
		})
	}
	return s.retriever.Retrieve(ctx, query, reEs8.WithFilters(esFilters))
}
