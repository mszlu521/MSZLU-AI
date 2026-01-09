package knowledges

import (
	"app/shared"
	"bufio"
	"bytes"
	"common/biz"
	"common/utils"
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

const (
	maxChildSize     = 500 //子块最大的长度
	childOverlapSize = 150 // 子块重叠的长度
)

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
	var parentModels []*model.DocumentChunk
	var childSchemaDocs []*schema.Document
	//这里我们先支持md文档
	if doc.FileType == ".md" {
		//md格式有清晰的标题 我们按照标题进行切分
		//documents = s.parseMarkdownHeaders(content)
		//if len(documents) == 0 {
		//	documents = append(documents, &schema.Document{
		//		ID:      doc.ID.String(),
		//		Content: content,
		//	})
		//}
		//对md文档进行层次性划分，我们以资料中提供的md文档为例子
		//h1认为是文档名称 h2认为是章节 h3认为是小节 进行层次性划分
		//chunk表中 存储h2的内容
		//提取标题 这里我们写个通用的
		h1Title := utils.ExtractTitle(content, "#")
		if h1Title == "" {
			h1Title = doc.Name
		}
		//获取h2的内容
		h2Block := utils.SplitByHeading(content, "##")
		for i, h2 := range h2Block {
			parentId := uuid.New()
			h2Title := utils.ExtractTitle(h2, "##")
			if h2Title == "" {
				h2Title = "概览"
			}
			//h2的内容是parent
			parentModels = append(parentModels, &model.DocumentChunk{
				BaseModel:       model.BaseModel{ID: parentId},
				DocumentID:      doc.ID,
				KnowledgeBaseID: kb.ID,
				Content:         h2,
				ChunkIndex:      i,
				MetaInfo: map[string]interface{}{
					"h1": h1Title,
					"h2": h2Title,
				},
				TokenCount: utils.GetTokenCount(h2),
				Status:     model.ChunkStatusEmbedded,
			})
			//获取h3的内容 这部分做为child
			h3Block := utils.SplitByHeading(h2, "###")
			for j, h3 := range h3Block {
				h3Title := utils.ExtractTitle(h3, "###")
				//这里我们给child的内容 添加一个前缀 表明所属的上级
				pathPrefix := fmt.Sprintf("【文档:%s】 > 【主题:%s】", h1Title, h2Title)
				if h3Title != "" {
					h3Title += " > 【子题: " + h3Title + "】"
				}
				//添加一个换行
				pathPrefix += "\n"
				//为了防止子内容过长，我们设定一个长度，做一次切分
				subTexts := utils.SplitTextByLength(h3, maxChildSize-len(pathPrefix), childOverlapSize)
				for k, text := range subTexts {
					//text就是最终子块的内容
					childSchemaDocs = append(childSchemaDocs, s.buildChildSchemaDoc(parentId, doc, kb, pathPrefix+text, i, j, k))
				}
			}
		}
	} else {
		//这个通用的处理，我们按照长度进行切分
		parentTexts := utils.SplitByWindow(content, 1200, 200)
		for i, pText := range parentTexts {
			parentModels = append(parentModels, &model.DocumentChunk{
				BaseModel:       model.BaseModel{ID: uuid.New()},
				DocumentID:      doc.ID,
				KnowledgeBaseID: kb.ID,
				Content:         pText,
				ChunkIndex:      i,
				MetaInfo: map[string]interface{}{
					"source":    doc.Name,
					"file_type": doc.FileType,
					"type":      "generic",
				},
				TokenCount: utils.GetTokenCount(pText),
				Status:     model.ChunkStatusEmbedded,
			})
			pathPrefix := fmt.Sprintf("【文档:%s】【片段:%d】\n", doc.Name, i+1)
			childTexts := utils.SplitByWindow(content, 400, 50)
			for j, cText := range childTexts {
				childSchemaDocs = append(childSchemaDocs, s.buildChildSchemaDoc(parentModels[i].ID, doc, kb, pathPrefix+cText, i, j, 0))
			}
		}
	}
	return s.saveToStores(ctx, kb, parentModels, childSchemaDocs)
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
				"doc_id.keyword": documentId.String(), //使用keyword精确匹配
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

const (
	maxSearchResult = 3 //设置一个最大搜索结果数量
)

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
	childDocs, err := retriever.Retrieve(ctx, params.Query)
	if err != nil {
		logs.Errorf("retrieve error: %v", err)
		return nil, biz.ErrRetriever
	}
	//我们需要查找匹配的子分段文档对应的父分段内容
	parentIdMap := make(map[string]float64) //doc_chunk_id:score
	var orderedParentIds []string
	for _, cd := range childDocs {
		pId, ok := cd.MetaData["parent_id"].(string)
		if !ok {
			continue
		}
		//记录pid childDocs是按照分数从高到低排序的 我们记录一下顺序
		if _, seen := parentIdMap[pId]; !seen {
			//后续如果父分段内容过多，我们只需要取前几个
			orderedParentIds = append(orderedParentIds, pId)
			parentIdMap[pId] = cd.Score()
		}
	}
	if len(orderedParentIds) == 0 {
		return &SearchResponse{
			KbId:  kbId,
			Query: params.Query,
			Took:  time.Since(startTime).Microseconds(),
			Total: 0,
		}, nil
	}
	if len(orderedParentIds) > maxSearchResult {
		//这里主要是为了防止知识库查询出来的内容过多，相似度太低的没有必要提供给大模型
		orderedParentIds = orderedParentIds[:maxSearchResult]
	}
	//获取父分段内容
	parentChunks, err := s.repo.getDocumentChunksByIds(ctx, orderedParentIds)
	if err != nil {
		logs.Errorf("get document chunks error: %v", err)
		return nil, errs.DBError
	}
	results := make([]*SearchResult, 0, len(parentChunks))
	for i, chunk := range parentChunks {
		results = append(results, &SearchResult{
			Content:    chunk.Content,
			DocumentId: chunk.DocumentID,
			Id:         chunk.ID,
			Metadata:   chunk.MetaInfo,
			Position:   i,
			Score:      parentIdMap[chunk.ID.String()],
		})
	}
	return &SearchResponse{
		KbId:    kbId,
		Query:   params.Query,
		Results: results,
		Took:    time.Since(startTime).Microseconds(),
		Total:   int64(len(results)),
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

func (s *service) buildChildSchemaDoc(parentId uuid.UUID, doc *model.Document, kb *model.KnowledgeBase, text string, i int, j int, k int) *schema.Document {
	return &schema.Document{
		ID:      uuid.New().String(),
		Content: text,
		MetaData: map[string]interface{}{
			"doc_id":    doc.ID.String(),
			"kb_id":     kb.ID.String(),
			"parent_id": parentId.String(),
			"seq":       fmt.Sprintf("%d.%d.%d", i, j, k),
		},
	}
}

func (s *service) saveToStores(ctx context.Context, kb *model.KnowledgeBase, parentModels []*model.DocumentChunk, docs []*schema.Document) error {
	//父分段直接存入数据库pg
	err := s.repo.createDocumentChunks(ctx, parentModels)
	if err != nil {
		logs.Errorf("create document chunks error: %v", err)
		return err
	}
	//子分段存入向量数据库，这里我们存入es中
	embedder, err := s.getEmbeddingConfig(kb.EmbeddingModelProvider, kb.EmbeddingModelName, kb.CreatorID)
	if err != nil {
		logs.Errorf("get embedding config error: %v", err)
		return biz.ErrEmbeddingConfigNotFound
	}
	indexer, err := es8.NewIndexer(ctx, &es8.IndexerConfig{
		Client: s.esClient,
		Index:  s.buildIndex(kb.ID),
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
		logs.Errorf("new indexer error: %v", err)
		return err
	}
	_, err = indexer.Store(ctx, docs)
	if err != nil {
		logs.Errorf("store documents error: %v", err)
		return err
	}
	return nil
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
