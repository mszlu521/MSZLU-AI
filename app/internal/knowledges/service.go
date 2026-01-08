package knowledges

import (
	"app/shared"
	"bufio"
	"bytes"
	"common/biz"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"model"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino-ext/components/indexer/es8"
	reES8 "github.com/cloudwego/eino-ext/components/retriever/es8"
	"github.com/cloudwego/eino-ext/components/retriever/es8/search_mode"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/schema"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/google/uuid"
	"github.com/mszlu521/thunder/ai/einos"
	"github.com/mszlu521/thunder/database"
	"github.com/mszlu521/thunder/errs"
	"github.com/mszlu521/thunder/event"
	"github.com/mszlu521/thunder/logs"
	"gorm.io/gorm"
)

type service struct {
	repo     repository
	esClient *elasticsearch.Client
}

func (s *service) createKnowledgeBase(ctx context.Context, userId uuid.UUID, req createKnowledgeBaseReq) (any, error) {
	kb := model.KnowledgeBase{
		BaseModel: model.BaseModel{
			ID: uuid.New(),
		},
		CreatorID:              userId,
		Name:                   req.Name,
		Description:            req.Description,
		ChatModelName:          req.ChatModelName,
		ChatModelProvider:      req.ChatModelProvider,
		EmbeddingModelName:     req.EmbeddingModelName,
		EmbeddingModelProvider: req.EmbeddingModelProvider,
		StorageType:            model.StorageTypeElasticSearch,
		StorageConfig:          model.JSON{},
		DocumentCount:          0,
		Tags:                   req.Tags,
	}
	err := s.repo.createKnowledgeBase(ctx, &kb)
	if err != nil {
		logs.Errorf("create knowledge base error: %v", err)
		return nil, errs.DBError
	}
	return &kb, nil
}

func (s *service) listKnowledgeBases(ctx context.Context, userId uuid.UUID, params listReq) (*ListResp, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	size := params.PageSize
	if size <= 0 {
		size = 10
	}
	filter := KnowledgeBaseFilter{
		Search: params.Search,
		Limit:  size,
		Offset: (page - 1) * size,
	}
	kbs, total, err := s.repo.listKnowledgeBases(ctx, userId, filter)
	if err != nil {
		logs.Errorf("list knowledge base error: %v", err)
		return nil, errs.DBError
	}
	return &ListResp{
		KnowledgeBases: kbs,
		Total:          total,
	}, nil
}

func (s *service) getKnowledgeBase(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*KnowledgeBaseResponse, error) {
	kb, err := s.repo.getKnowledgeBase(ctx, userId, id)
	if err != nil {
		logs.Errorf("get knowledge base error: %v", err)
		return nil, errs.DBError
	}
	//统计文档数和总字节数
	totalSize, docCount, err := s.repo.countKnowledgeBaseDocuments(ctx, kb.ID)
	if err != nil {
		logs.Errorf("count knowledge base documents error: %v", err)
		return nil, errs.DBError
	}
	return &KnowledgeBaseResponse{
		Id:                     kb.ID,
		Name:                   kb.Name,
		Description:            kb.Description,
		EmbeddingModelName:     kb.EmbeddingModelName,
		EmbeddingModelProvider: kb.EmbeddingModelProvider,
		ChatModelName:          kb.ChatModelName,
		ChatModelProvider:      kb.ChatModelProvider,
		StorageType:            kb.StorageType,
		StorageConfig:          kb.StorageConfig,
		Tags:                   kb.Tags,
		TotalSize:              totalSize,
		DocumentCount:          int(docCount),
		CreatorId:              kb.CreatorID,
		CreatedAt:              kb.CreatedAt.Unix(),
		UpdatedAt:              kb.UpdatedAt.Unix(),
	}, nil
}

func (s *service) updateKnowledgeBase(ctx context.Context, userId uuid.UUID, id uuid.UUID, req updateKnowledgeBaseReq) (any, error) {
	kb, err := s.repo.getKnowledgeBase(ctx, userId, id)
	if err != nil {
		logs.Errorf("get knowledge base error: %v", err)
		return nil, errs.DBError
	}
	if kb == nil {
		return nil, biz.ErrKnowledgeBaseNotFound
	}
	if req.Name != "" {
		kb.Name = req.Name
	}
	if req.Description != "" {
		kb.Description = req.Description
	}
	if req.EmbeddingModelName != "" {
		kb.EmbeddingModelName = req.EmbeddingModelName
	}
	if req.EmbeddingModelProvider != "" {
		kb.EmbeddingModelProvider = req.EmbeddingModelProvider
	}
	if req.ChatModelName != "" {
		kb.ChatModelName = req.ChatModelName
	}
	if req.ChatModelProvider != "" {
		kb.ChatModelProvider = req.ChatModelProvider
	}
	err = s.repo.updateKnowledgeBase(ctx, kb)
	if err != nil {
		logs.Errorf("update knowledge base error: %v", err)
		return nil, errs.DBError
	}
	return kb, nil
}

func (s *service) deleteKnowledgeBase(ctx context.Context, userId uuid.UUID, id uuid.UUID) error {
	kb, err := s.repo.getKnowledgeBase(ctx, userId, id)
	if err != nil {
		logs.Errorf("get knowledge base error: %v", err)
		return errs.DBError
	}
	if kb == nil {
		return biz.ErrKnowledgeBaseNotFound
	}
	err = s.repo.deleteKnowledgeBase(ctx, kb.ID)
	if err != nil {
		logs.Errorf("delete knowledge base error: %v", err)
		return errs.DBError
	}
	return nil
}

func (s *service) listDocuments(ctx context.Context, userId uuid.UUID, kbId uuid.UUID, params listDocumentReq) (*ListDocumentsResp, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	size := params.PageSize
	if size <= 0 {
		size = 10
	}
	filter := DocumentFilter{
		Status: params.Status,
		Search: params.Search,
		Limit:  size,
		Offset: (page - 1) * size,
	}
	documents, total, err := s.repo.listDocuments(ctx, userId, kbId, filter)
	if err != nil {
		logs.Errorf("list documents error: %v", err)
		return nil, errs.DBError
	}
	return &ListDocumentsResp{
		Documents: documents,
		Total:     total,
	}, nil
}

func (s *service) uploadDocuments(ctx context.Context, userId uuid.UUID, kbId uuid.UUID, uploadFile *multipart.FileHeader) (any, error) {
	//读取文件信息，创建Document对象
	//读取文件内容，进行向量化和索引，其中要进行切分，切分后的数据存入documentchunk表中
	//同时将切分后的内容，向量化后存入向量数据库中
	//文件可以存入云存储中
	//我们先写读取文件信息，创建Document对象，并存入数据库中这个步骤
	//先检查知识库是否存在
	kb, err := s.repo.getKnowledgeBase(ctx, userId, kbId)
	if err != nil {
		logs.Errorf("get knowledge base error: %v", err)
		return nil, errs.DBError
	}
	if kb == nil {
		return nil, biz.ErrKnowledgeBaseNotFound
	}
	selectParser := parser.TextParser{}
	//读取文件内容
	loader, err := file.NewFileLoader(ctx, &file.FileLoaderConfig{
		Parser: selectParser,
	})
	if err != nil {
		logs.Errorf("new file loader error: %v", err)
		return nil, biz.FileLoadError
	}
	src, err := uploadFile.Open()
	if err != nil {
		logs.Errorf("open file error: %v", err)
		return nil, biz.FileLoadError
	}
	defer src.Close()
	tempFile, err := s.createTempFileFromUploadFile(src, uploadFile.Filename)
	if err != nil {
		logs.Errorf("create temp file error: %v", err)
		return nil, biz.FileLoadError
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())
	//这个URL是文件的地址，正常我们应该上传到云存储中，这里我们先创建一个本地临时文件来获取内容
	docs, err := loader.Load(ctx, document.Source{
		URI: tempFile.Name(),
	})
	if err != nil {
		logs.Errorf("load file error: %v", err)
		return nil, biz.FileLoadError
	}
	//文件后轴
	ext := strings.ToLower(filepath.Ext(uploadFile.Filename))
	doc := &model.Document{
		KnowledgeBaseID: kb.ID,
		CreatorID:       userId,
		Name:            uploadFile.Filename,
		FileType:        ext,
		Size:            uploadFile.Size,
		StorageKey:      uploadFile.Filename,
		FileHash:        "",
		Status:          model.DocumentStatusPending,
		ErrorMessage:    "",
	}
	err = s.repo.createDocument(ctx, doc)
	if err != nil {
		logs.Errorf("create document error: %v", err)
		return nil, errs.DBError
	}
	//对文件内容的处理，切分+向量化+索引 放入go协程中，进行处理过程比较长
	go func() {
		//做状态更新
		//这个执行时间长，不能用上面的上下文
		ctx = context.Background()
		err = s.repo.updateDocumentStatus(ctx, doc.ID, model.DocumentStatusProcessing)
		if err != nil {
			logs.Errorf("update document status error: %v", err)
			return
		}
		//处理文件
		err = s.processDocumentAndVectorAndStore(ctx, doc, docs, kb)
		if err != nil {
			logs.Errorf("process file error: %v", err)
			//更新状态为失败
			err = s.repo.updateDocumentStatus(ctx, doc.ID, model.DocumentStatusFailed)
			if err != nil {
				logs.Errorf("update document status error: %v", err)
				return
			}
			return
		}
		//最后更新状态
		err = s.repo.updateDocumentStatus(ctx, doc.ID, model.DocumentStatusCompleted)
		if err != nil {
			logs.Errorf("update document status error: %v", err)
			return
		}
	}()
	return doc, nil
}

func (s *service) processDocumentAndVectorAndStore(ctx context.Context, doc *model.Document, docs []*schema.Document, kb *model.KnowledgeBase) error {
	//获取文档内容
	var content string
	if len(docs) > 0 && docs[0] != nil {
		content = docs[0].Content
	}
	//如果文档内容为空 直接返回
	if content == "" {
		logs.Warnf("document content is empty")
		return nil
	}
	//接下来就是切分+向量化+索引，我们先不考虑切分，直接存储全部的内容
	var documents []*schema.Document
	//这里我们先支持md文档
	if doc.FileType == ".md" {
		//md格式有清晰的标题 我们按照标题进行切分
		documents = s.parseMarkdownHeaders(content)
		if len(documents) == 0 {
			documents = append(documents, &schema.Document{
				ID:      doc.ID.String(),
				Content: content,
			})
		}
	} else {
		//这里我们假设我们做了切分
		documents = append(documents, &schema.Document{
			ID:      doc.ID.String(),
			Content: content,
		})
	}
	//上述切分在元数据中生成了h1,h2,h3的元数据，我们可以将这些设置为标题 章节 段落，在生成文本的时候，将这些信息添加进去，方便知道上下文
	var chunkMetaData []map[string]interface{}
	for _, d := range documents {
		metadata := map[string]interface{}{}
		//可以给元数据加一些文档的信息, 如果要进行过滤，这些元数据可以作为过滤条件
		metadata["doc_name"] = doc.Name
		metadata["file_type"] = doc.FileType
		if d.MetaData != nil {
			if v, ok := d.MetaData["h1"]; ok {
				metadata["title"] = v
			}
			if v, ok := d.MetaData["h2"]; ok {
				metadata["chapter"] = v
			}
			if v, ok := d.MetaData["h3"]; ok {
				metadata["section"] = v
			}
		}
		chunkMetaData = append(chunkMetaData, metadata)
	}
	//我们给内容 添加一些前缀信息，就是如果有title chapter section这些，我们将这些信息添加上去,让内容知道是哪个模块下的
	var chunks []string
	for i, d := range documents {
		var prefixBuilder strings.Builder
		if len(chunkMetaData) > i {
			meta := chunkMetaData[i]
			if v, ok := meta["title"].(string); ok && v != "" {
				prefixBuilder.WriteString(fmt.Sprintf("[%s] \n", v))
			}
			if v, ok := meta["chapter"].(string); ok && v != "" {
				prefixBuilder.WriteString(fmt.Sprintf("章节： %s \n", v))
			}
			if v, ok := meta["section"].(string); ok && v != "" {
				prefixBuilder.WriteString(fmt.Sprintf("小节：%s \n", v))
			}
		}
		//添加前缀
		chunks = append(chunks, prefixBuilder.String()+d.Content)
	}
	if len(chunks) == 0 {
		//如果没有任何内容 就返回默认的
		chunks = []string{content}
		chunkMetaData = []map[string]interface{}{
			{
				"doc_name":  doc.Name,
				"file_type": doc.FileType,
			},
		}
	}
	embedder, err := s.getEmbeddingConfig(kb.EmbeddingModelProvider, kb.EmbeddingModelName, kb.CreatorID)
	//接下来我们存储 用到了eino
	indexer, err := es8.NewIndexer(ctx, &es8.IndexerConfig{
		Client: s.esClient,
		Index:  s.buildIndex(kb.ID),
		DocumentToFields: func(ctx context.Context, docs *schema.Document) (field2Value map[string]es8.FieldValue, err error) {
			return map[string]es8.FieldValue{
				"content": {
					Value:    docs.Content,
					EmbedKey: "content_vector",
				},
				"doc_id": {
					Value: doc.ID.String(),
				},
				"kb_id": {
					Value: kb.ID.String(),
				},
				"metadata": {
					Value: docs.MetaData,
				},
			}, nil
		},
		Embedding: embedder,
	})
	if err != nil {
		logs.Errorf("new indexer error: %v", err)
		return err
	}
	var schemaDocs []*schema.Document
	//我们存储一些元数据进去，便于后续搜索
	for i, chunk := range chunks {
		//这个地方我们添加一些比如chunk的索引，方便后续搜索
		currentChunkMeta := chunkMetaData[i%len(chunkMetaData)]
		currentChunkMeta["position"] = i
		//后续也可以加一些比如标题名称 作者名称 文章类型 等等的
		schemaDocs = append(schemaDocs, &schema.Document{
			ID:       uuid.New().String(),
			Content:  chunk,
			MetaData: currentChunkMeta,
		})
	}
	ids, err := indexer.Store(ctx, schemaDocs)
	if err != nil {
		logs.Errorf("store documents error: %v", err)
		return err
	}
	//接下来我们将其也存储一份到数据库中，做为原始数据凭证
	var docChunks []*model.DocumentChunk
	for i, v := range schemaDocs {
		chunk := &model.DocumentChunk{
			BaseModel: model.BaseModel{
				ID: uuid.New(),
			},
			DocumentID:      doc.ID,
			KnowledgeBaseID: kb.ID,
			Content:         v.Content,
			ChunkIndex:      i,
			TokenCount:      len(v.Content), //这个后续我们获取
			MetaInfo:        v.MetaData,
			Status:          model.ChunkStatusEmbedded,
		}
		if i < len(ids) {
			chunk.ElasticSearchID = ids[i]
		}
		docChunks = append(docChunks, chunk)
	}
	err = s.repo.createDocumentChunks(ctx, docChunks)
	if err != nil {
		logs.Errorf("create document chunks error: %v", err)
		return err
	}
	return nil
}

func (s *service) createTempFileFromUploadFile(src multipart.File, fileName string) (*os.File, error) {
	//创建一个临时文件
	tempFile, err := os.CreateTemp("", "upload-*.tmp")
	if err != nil {
		return nil, err
	}
	defer tempFile.Close()
	//复制文件内容到临时文件
	_, err = io.Copy(tempFile, src)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}
	//重置文件指针到开始位置
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}
	return tempFile, nil
}

func (s *service) getEmbeddingConfig(provider string, embeddingModelName string, creatorID uuid.UUID) (embedding.Embedder, error) {
	trigger, err := event.Trigger("getEmbeddingConfig", &shared.LLMParams{
		Provider:  provider,
		Model:     embeddingModelName,
		UserId:    creatorID,
		ModelType: model.LLMTypeEmbedding,
	})
	if err != nil {
		return nil, err
	}
	response := trigger.(*shared.EmbeddingConfigResponse)
	embedder, err := einos.LoadEmbedding(context.Background(),
		response.Model.ProviderConfig.Provider,
		response.Model.ToEmbeddingConfig())
	return embedder, err
}

func (s *service) deleteDocuments(ctx context.Context, userId uuid.UUID, kbId uuid.UUID, documentId uuid.UUID) error {
	//先确认参数正确
	knowledgeBase, err := s.repo.getKnowledgeBase(ctx, userId, kbId)
	if err != nil {
		logs.Errorf("get knowledge base error: %v", err)
		return errs.DBError
	}
	if knowledgeBase == nil {
		return biz.ErrKnowledgeBaseNotFound
	}
	doc, err := s.repo.getDocument(ctx, userId, kbId, documentId)
	if err != nil {
		logs.Errorf("get document error: %v", err)
		return errs.DBError
	}
	if doc == nil {
		return biz.ErrDocumentNotFound
	}
	//删除多个表的数据，所以这里必须要用事务
	err = s.repo.transaction(ctx, func(tx *gorm.DB) error {
		//先删除文档
		err = s.repo.deleteDocuments(ctx, tx, userId, kbId, documentId)
		if err != nil {
			logs.Errorf("delete documents error: %v", err)
			return err
		}
		//删除文档片段
		err = s.repo.deleteDocumentChunks(ctx, tx, kbId, documentId)
		if err != nil {
			logs.Errorf("delete document chunks error: %v", err)
			return err
		}
		//删除es的索引
		if knowledgeBase.StorageType == model.StorageTypeElasticSearch {
			err = s.deleteEsIndex(ctx, kbId, documentId)
			if err != nil {
				logs.Errorf("delete es index error: %v", err)
				return err
			}
		}
		return nil
	})
	if err != nil {
		logs.Errorf("delete documents error: %v", err)
		return errs.DBError
	}
	return nil
}

func (s *service) deleteEsIndex(ctx context.Context, kbId uuid.UUID, documentId uuid.UUID) error {
	index := s.buildIndex(kbId)
	//需要删除doc_id这个字段匹配的文档
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"doc_id,keyword": documentId.String(), //使用keyword精确匹配
			},
		},
	}
	//查询条件转换为json
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	err := encoder.Encode(query)
	if err != nil {
		logs.Errorf("encode query error: %v", err)
		return err
	}
	res, err := s.esClient.DeleteByQuery(
		[]string{index},
		&buf,
		s.esClient.DeleteByQuery.WithContext(ctx),
	)
	if err != nil {
		logs.Errorf("delete by query error: %v", err)
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		logs.Errorf("delete by query error: %v", res)
		return err
	}
	return nil
}

func (s *service) buildIndex(kbId uuid.UUID) string {
	return fmt.Sprintf("kb_%s", kbId.String())
}

func (s *service) searchKnowledgeBase(ctx context.Context, userId uuid.UUID, kbId uuid.UUID, params searchParams) (*SearchResponse, error) {
	//记录开始时间
	startTime := time.Now()
	index := s.buildIndex(kbId)
	//验证知识库是否存在
	knowledgeBase, err := s.repo.getKnowledgeBase(ctx, userId, kbId)
	if err != nil {
		logs.Errorf("get knowledge base error: %v", err)
		return nil, errs.DBError
	}
	if knowledgeBase == nil {
		return nil, biz.ErrKnowledgeBaseNotFound
	}
	if params.Query == "" {
		//返回空结果
		return &SearchResponse{
			KbId:    kbId,
			Query:   params.Query,
			Results: []*SearchResult{},
			Took:    time.Since(startTime).Microseconds(),
			Total:   0,
		}, nil
	}
	//获取到向量模型配置
	embedder, err := s.getEmbeddingConfig(knowledgeBase.EmbeddingModelProvider, knowledgeBase.EmbeddingModelName, userId)
	if err != nil {
		logs.Errorf("get embedding config error: %v", err)
		return nil, biz.ErrEmbeddingConfigNotFound
	}
	//构建es的检索
	retriever, err := reES8.NewRetriever(ctx, &reES8.RetrieverConfig{
		Client: s.esClient,
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
			doc.MetaData["doc_id"] = src["doc_id"]
			doc.MetaData["kb_id"] = src["kb_id"]
			if hit.Score_ != nil {
				doc.WithScore(float64(*hit.Score_))
			}
			return doc, nil
		},
		Embedding: embedder,
	})
	if err != nil {
		logs.Errorf("new retriever error: %v", err)
		return nil, biz.ErrRetriever
	}
	docs, err := retriever.Retrieve(ctx, params.Query)
	if err != nil {
		logs.Errorf("retrieve error: %v", err)
		return nil, biz.ErrRetriever
	}
	results := make([]*SearchResult, len(docs))
	for i, doc := range docs {
		docId := doc.MetaData["doc_id"].(string)
		docIdUUID := uuid.MustParse(docId)
		documentContent, err := s.repo.getDocument(ctx, userId, kbId, docIdUUID)
		if err != nil {
			logs.Errorf("get document error: %v", err)
			return nil, biz.ErrDocumentNotFound
		}
		floatPosition := doc.MetaData["position"].(float64)
		result := &SearchResult{
			Content:    doc.Content,
			DocumentId: docIdUUID,
			Id:         uuid.MustParse(doc.ID),
			Metadata:   doc.MetaData,
			Position:   int(floatPosition),
			Score:      doc.Score(),
			Document:   documentContent,
		}
		results[i] = result
	}
	return &SearchResponse{
		KbId:    kbId,
		Query:   params.Query,
		Results: results,
		Took:    time.Since(startTime).Microseconds(),
		Total:   int64(len(docs)),
	}, nil
}

func (s *service) parseMarkdownHeaders(content string) []*schema.Document {
	var docs []*schema.Document
	scanner := bufio.NewScanner(strings.NewReader(content))
	// 正则匹配 #, ##, ### (最多支持到 h6)
	// 格式: 行首 + 1-6个# + 空格 + 标题内容
	headerRegex := regexp.MustCompile(`^(#{1,6})\s+(.*)`)
	var currentBuffer strings.Builder
	// 记录当前的标题层级状态 {"h1": "标题A", "h2": "子标题B"}
	currentHeaders := make(map[string]string)

	flushBuffer := func() {
		text := strings.TrimSpace(currentBuffer.String())
		if text == "" {
			return
		}

		// 深拷贝当前的 header 状态，防止被后续修改影响
		meta := make(map[string]interface{})
		for k, v := range currentHeaders {
			meta[k] = v
		}

		docs = append(docs, &schema.Document{
			Content:  text,
			MetaData: meta,
		})
		currentBuffer.Reset()
	}

	for scanner.Scan() {
		line := scanner.Text()
		matches := headerRegex.FindStringSubmatch(line)

		if len(matches) == 3 {
			// === 发现新标题 ===

			// 1. 如果缓冲区有上一段的内容，先保存上一段
			flushBuffer()

			// 2. 更新层级上下文
			hashes := matches[1]                       // "##"
			titleText := strings.TrimSpace(matches[2]) // "部署指南"
			level := len(hashes)                       // 2

			// 记录当前级别标题
			levelKey := fmt.Sprintf("h%d", level)
			currentHeaders[levelKey] = titleText

			// 清除比当前级别更深的标题 (例如遇到新的 h2，旧的 h3, h4 应该失效)
			for i := level + 1; i <= 6; i++ {
				delete(currentHeaders, fmt.Sprintf("h%d", i))
			}
			// 3. 将标题行本身也写入新缓冲区的开头 (可选项，有助于语义完整)
			currentBuffer.WriteString(line + "\n")
		} else {
			// === 普通正文 ===
			currentBuffer.WriteString(line + "\n")
		}
	}
	// 处理最后剩余的内容
	flushBuffer()

	return docs
}
func newService() *service {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			"http://localhost:9200",
		},
		Username: "elastic",
		Password: "mszlu123456!@#$",
	})
	if err != nil {
		panic(err)
	}
	return &service{
		repo:     newModels(database.GetPostgresDB().GormDB),
		esClient: client,
	}
}
