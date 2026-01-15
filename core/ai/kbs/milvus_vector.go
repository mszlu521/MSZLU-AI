package kbs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/indexer/milvus"
	reMilvus "github.com/cloudwego/eino-ext/components/retriever/milvus"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/mszlu521/thunder/logs"
)

const dim = 768 //要和向量模型保持一致 要不然查询不出来

type MilvusVectorStore struct {
	indexer   *milvus.Indexer
	retriever *reMilvus.Retriever
}

func NewMilvusVectorStore(
	ctx context.Context,
	c client.Client,
	collectionName string,
	embedder embedding.Embedder,
) (*MilvusVectorStore, error) {
	//先创建collection
	err := ensureMilvusCollection(ctx, c, collectionName)
	if err != nil {
		return nil, err
	}
	//创建eino的milvus indexer
	indexer, err := milvus.NewIndexer(ctx, &milvus.IndexerConfig{
		Client:     c,
		Collection: collectionName,
		MetricType: milvus.COSINE,
		DocumentConverter: func(ctx context.Context, docs []*schema.Document, vectors [][]float64) ([]interface{}, error) {
			rows := make([]interface{}, len(docs))
			for i, doc := range docs {
				vec32 := make([]float32, len(vectors[i]))
				for j, v := range vectors[i] {
					vec32[j] = float32(v)
				}
				rows[i] = map[string]interface{}{
					"id":        doc.ID,
					"parent_id": doc.MetaData["parent_id"],
					"doc_id":    doc.MetaData["doc_id"],
					"content":   doc.Content,
					"vector":    vec32,
					"metadata":  doc.MetaData,
				}
			}
			return rows, nil
		},
		Fields: []*entity.Field{
			{
				Name:     "id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "128",
				},
			},
			{
				Name:     "parent_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "128",
				},
			},
			{
				Name:     "doc_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "128",
				},
			},
			{
				Name:     "content",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "8192",
				},
			},
			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", dim),
				},
			},
			{
				Name:     "metadata",
				DataType: entity.FieldTypeJSON,
			},
		},
		Embedding: embedder,
	})
	if err != nil {
		logs.Errorf("create indexer error: %v", err)
		return nil, err
	}
	//创建eino的milvus retriever
	param, err := entity.NewIndexHNSWSearchParam(64)
	if err != nil {
		return nil, err
	}
	retriever, err := reMilvus.NewRetriever(ctx, &reMilvus.RetrieverConfig{
		Client:      c,
		Collection:  collectionName,
		MetricType:  entity.COSINE,
		VectorField: "vector",
		OutputFields: []string{
			"id",
			"content",
			"metadata",
		},
		Sp: param,
		VectorConverter: func(ctx context.Context, vectors [][]float64) ([]entity.Vector, error) {
			res := make([]entity.Vector, len(vectors))
			for i, vector := range vectors {
				fv := make([]float32, len(vector))
				for j := range vector {
					fv[j] = float32(vector[j])
				}
				res[i] = entity.FloatVector(fv)
			}
			return res, nil
		},
		DocumentConverter: func(ctx context.Context, result client.SearchResult) ([]*schema.Document, error) {
			docs := make([]*schema.Document, 0)
			//获取各字段数据
			idColumn, ok := result.Fields.GetColumn("id").(*entity.ColumnVarChar)
			if !ok {
				return nil, fmt.Errorf("id column not found")
			}
			contentColumn, ok := result.Fields.GetColumn("content").(*entity.ColumnVarChar)
			if !ok {
				return nil, fmt.Errorf("content column not found")
			}
			metadataColumn, ok := result.Fields.GetColumn("metadata").(*entity.ColumnJSONBytes)
			if !ok {
				return nil, fmt.Errorf("metadata column not found")
			}
			//构建文档列表
			for i := 0; i < result.ResultCount; i++ {
				doc := &schema.Document{}
				id, err := idColumn.ValueByIdx(i)
				if err != nil {
					continue
				}
				doc.ID = id
				content, err := contentColumn.ValueByIdx(i)
				if err != nil {
					continue
				}
				doc.Content = content
				metadataStr, err := metadataColumn.ValueByIdx(i)
				if err != nil {
					continue
				}
				var metadata map[string]interface{}
				err = json.Unmarshal(metadataStr, &metadata)
				if err != nil {
					metadata = make(map[string]interface{})
				}
				doc.MetaData = metadata
				//设置分数
				if i < len(result.Scores) {
					doc.WithScore(float64(result.Scores[i]))
				}
				docs = append(docs, doc)
			}
			return docs, nil
		},
		Embedding: embedder,
	})
	if err != nil {
		logs.Errorf("create retriever error: %v", err)
		return nil, err
	}
	return &MilvusVectorStore{
		indexer:   indexer,
		retriever: retriever,
	}, nil
}
func (s *MilvusVectorStore) Store(ctx context.Context, docs []*schema.Document) error {
	//这里分批插入
	const batchSize = 50
	total := len(docs)
	if total == 0 {
		return nil
	}
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		_, err := s.indexer.Store(ctx, docs[i:end])
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *MilvusVectorStore) Search(ctx context.Context, query string, topK int, filters SearchFilter) ([]*schema.Document, error) {
	//构建milvus filter
	var expr string
	if len(filters) > 0 {
		expr = s.buildMilvusFilter(filters)
	}
	options := []retriever.Option{
		retriever.WithTopK(topK),
	}
	if expr != "" {
		options = append(options, reMilvus.WithFilter(expr))
	}
	return s.retriever.Retrieve(ctx, query, options...)
}

func (s *MilvusVectorStore) buildMilvusFilter(filters SearchFilter) string {
	expr := make([]string, 0)
	for key, value := range filters {
		switch v := value.(type) {
		case string:
			expr = append(expr, fmt.Sprintf("metadata['%s'] == '%s'", key, v))
		case int, int32, int64, uint, uint32, uint64:
			expr = append(expr, fmt.Sprintf("metadata['%s'] == %d", key, v))
		case float32, float64:
			expr = append(expr, fmt.Sprintf("metadata['%s'] == %f", key, v))
		case bool:
			expr = append(expr, fmt.Sprintf("metadata['%s'] == %t", key, v))
		}
	}
	if len(expr) == 0 {
		return ""
	}
	if len(expr) == 1 {
		return expr[0]
	}
	//多个条件就用and连接
	result := expr[0]
	for i := 1; i < len(expr); i++ {
		result += " AND " + expr[i]
	}
	logs.Infof("milvus filter: %s", result)
	return result
}
func ensureMilvusCollection(ctx context.Context, client client.Client, collectionName string) error {
	//先判断collection是否存在
	has, err := client.HasCollection(ctx, collectionName)
	if err != nil {
		return err
	}
	if has {
		//如果已经存在 就直接返回就行
		return nil
	}
	collectionSchema := &entity.Schema{
		CollectionName: collectionName,
		AutoID:         true,
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeVarChar,
				PrimaryKey: true,
				TypeParams: map[string]string{
					"max_length": "128",
				},
			},
			{
				Name:     "parent_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "128",
				},
			},
			{
				Name:     "doc_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "128",
				},
			},
			{
				Name:     "content",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "8192",
				},
			},
			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", dim),
				},
			},
			{
				Name:     "metadata",
				DataType: entity.FieldTypeJSON,
				TypeParams: map[string]string{
					"max_length": "4096",
				},
			},
		},
	}
	err = client.CreateCollection(ctx, collectionSchema, 2)
	if err != nil {
		return err
	}
	//创建向量索引
	hnswIndex, err := entity.NewIndexHNSW(entity.COSINE, 16, 200)
	if err != nil {
		return err
	}
	err = client.CreateIndex(ctx, collectionName, "vector", hnswIndex, false)
	if err != nil {
		return err
	}
	err = client.LoadCollection(ctx, collectionName, false)
	return err
}
