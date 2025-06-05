# LangChain 整合系統

這個包實現了 Assistant 與 LangChain-Go 的深度整合，提供完整的 AI 工作流支援和高級 RAG 功能。

## 架構概述

LangChain 整合系統採用模塊化設計，支援多種 AI 工作流：

```
langchain/
├── client.go            # LangChain 客戶端
├── service.go           # LangChain 服務層
├── adapter.go           # Assistant 適配器
├── agents/              # 智能代理系統
│   ├── base.go          # 代理基礎類
│   ├── development.go   # 開發專家代理
│   ├── database.go      # 資料庫專家代理
│   ├── infrastructure.go # 基礎設施代理
│   └── research.go      # 研究代理
├── chains/              # AI 工作流鏈
│   ├── base.go          # 鏈基礎類
│   ├── sequential.go    # 順序執行鏈
│   ├── parallel.go      # 並行執行鏈
│   ├── conditional.go   # 條件執行鏈
│   ├── rag.go           # RAG 鏈
│   └── rag_enhanced.go  # 增強 RAG 鏈
├── memory/              # 記憶系統
│   ├── manager.go       # 記憶管理器
│   ├── shortterm.go     # 短期記憶
│   ├── longterm.go      # 長期記憶
│   └── personalization.go # 個人化記憶
├── vectorstore/         # 向量存儲
│   └── pgvector.go      # PostgreSQL 向量存儲
└── documentloader/      # 文檔處理
    └── loader.go        # 多格式文檔加載器
```

## 設計理念

### 🤖 AI 原生集成
LangChain 整合系統原生支援 AI 工作流：
- **原生 LLM 支援**: 直接使用 LangChain 的 LLM 抽象
- **工作流編排**: 用 Chains 建構複雜 AI 工作流
- **向量檢索**: 內建語義搜尋和 RAG 功能
- **記憶管理**: 長短期記憶的智能管理

### 🔗 無縫橋接
與 Assistant 核心系統的完美整合：
- **雙向兼容**: 支援 LangChain 和原生 Assistant API
- **為寮倒降**: 自動在 LangChain 和原生實現間切換
- **配置驅動**: 通過配置控制使用優先級
- **狀態同步**: 保持記憶和上下文的一致性

### 🚀 高級 RAG
企業級的檢索增強生成：
- **多模態檢索**: 支援文本、代碼、圖像等格式
- **智能分割**: 自適應文檔分割策略
- **層次檢索**: 實現粗精度的層次檢索
- **上下文偏重**: 根據上下文調整檢索策略

## 核心介面

### LangChain 客戶端

```go
// LangChainClient 提供 LangChain 功能的統一入口
type LangChainClient interface {
    // 基礎 LLM 功能
    GenerateText(ctx context.Context, prompt string, options *GenerateOptions) (*GenerateResponse, error)
    
    // 工作流鏈執行
    RunChain(ctx context.Context, chainName string, inputs map[string]any) (*ChainResult, error)
    
    // 智能代理交互
    InvokeAgent(ctx context.Context, agentName string, input *AgentInput) (*AgentResponse, error)
    
    // 向量存儲操作
    StoreDocuments(ctx context.Context, docs []Document) error
    SearchSimilar(ctx context.Context, query string, opts *SearchOptions) ([]Document, error)
    
    // 記憶管理
    StoreMemory(ctx context.Context, sessionID string, memory *Memory) error
    RetrieveMemory(ctx context.Context, sessionID string) (*Memory, error)
    
    // 生命週期管理
    Initialize(ctx context.Context, config *Config) error
    Close() error
}

// LangChainClientImpl 實現 LangChain 客戶端
type LangChainClientImpl struct {
    // LangChain 組件
    llm          llms.Model
    embeddings   embeddings.Embedder
    vectorStore  vectorstores.VectorStore
    
    // 智能代理
    agents       map[string]agents.Agent
    agentExecutor *agents.Executor
    
    // 工作流鏈
    chains       map[string]chains.Chain
    
    // 記憶系統
    memory       schema.Memory
    
    // 配置和日誌
    config       *Config
    logger       *slog.Logger
}

func NewLangChainClient(config *Config, logger *slog.Logger) (LangChainClient, error) {
    client := &LangChainClientImpl{
        agents: make(map[string]agents.Agent),
        chains: make(map[string]chains.Chain),
        config: config,
        logger: logger,
    }
    
    // 初始化 LLM
    llm, err := client.initializeLLM(config.LLM)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize LLM: %w", err)
    }
    client.llm = llm
    
    // 初始化嵌入模型
    embeddings, err := client.initializeEmbeddings(config.Embeddings)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize embeddings: %w", err)
    }
    client.embeddings = embeddings
    
    // 初始化向量存儲
    vectorStore, err := client.initializeVectorStore(config.VectorStore)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize vector store: %w", err)
    }
    client.vectorStore = vectorStore
    
    // 初始化智能代理
    err = client.initializeAgents(config.Agents)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize agents: %w", err)
    }
    
    // 初始化工作流鏈
    err = client.initializeChains(config.Chains)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize chains: %w", err)
    }
    
    return client, nil
}
```

### 智能代理系統

```go
// LangChainAgent 在 LangChain 中的智能代理實現
type LangChainAgent struct {
    name        string
    description string
    
    // LangChain 組件
    llm         llms.Model
    tools       []tools.Tool
    memory      schema.Memory
    
    // Assistant 集成
    assistantAgent core.Agent
    
    // 執行器
    executor    *agents.Executor
    
    // 配置
    config      *AgentConfig
    logger      *slog.Logger
}

func NewDevelopmentAgent(config *AgentConfig, logger *slog.Logger) (*LangChainAgent, error) {
    agent := &LangChainAgent{
        name:        "development-expert",
        description: "Specialized agent for software development tasks",
        config:      config,
        logger:      logger,
    }
    
    // 初始化 LLM
    llm, err := initializeLLM(config.LLM)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize LLM: %w", err)
    }
    agent.llm = llm
    
    // 初始化開發工具
    devTools, err := agent.initializeDevelopmentTools()
    if err != nil {
        return nil, fmt.Errorf("failed to initialize development tools: %w", err)
    }
    agent.tools = devTools
    
    // 初始化記憶
    memory := memory.NewBufferMemory()
    agent.memory = memory
    
    // 創建 Agent 執行器
    executor, err := agents.NewExecutor(
        agents.NewReActAgent(
            llm,
            devTools,
            agents.WithMemory(memory),
            agents.WithMaxIterations(10),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create agent executor: %w", err)
    }
    agent.executor = executor
    
    return agent, nil
}

func (la *LangChainAgent) ProcessTask(ctx context.Context, task *Task) (*TaskResult, error) {
    // 準備輸入
    inputs := map[string]any{
        "input": task.Description,
        "context": task.Context,
    }
    
    // 執行 Agent
    result, err := la.executor.Call(ctx, inputs)
    if err != nil {
        return nil, fmt.Errorf("agent execution failed: %w", err)
    }
    
    // 解析結果
    output, ok := result["output"].(string)
    if !ok {
        return nil, fmt.Errorf("invalid agent output format")
    }
    
    taskResult := &TaskResult{
        Output:      output,
        Actions:     la.extractActions(result),
        Reasoning:   la.extractReasoning(result),
        Confidence:  la.calculateConfidence(result),
        Metadata:    result,
    }
    
    return taskResult, nil
}

func (la *LangChainAgent) initializeDevelopmentTools() ([]tools.Tool, error) {
    var devTools []tools.Tool
    
    // 代碼分析工具
    codeAnalyzer, err := tools.NewCodeAnalyzer()
    if err != nil {
        return nil, fmt.Errorf("failed to create code analyzer: %w", err)
    }
    devTools = append(devTools, codeAnalyzer)
    
    // Git 操作工具
    gitTool, err := tools.NewGitTool()
    if err != nil {
        return nil, fmt.Errorf("failed to create git tool: %w", err)
    }
    devTools = append(devTools, gitTool)
    
    // 文件系統工具
    fileSystemTool, err := tools.NewFileSystemTool()
    if err != nil {
        return nil, fmt.Errorf("failed to create filesystem tool: %w", err)
    }
    devTools = append(devTools, fileSystemTool)
    
    // 編譯和測試工具
    buildTool, err := tools.NewBuildTool()
    if err != nil {
        return nil, fmt.Errorf("failed to create build tool: %w", err)
    }
    devTools = append(devTools, buildTool)
    
    return devTools, nil
}
```

### RAG 增強鏈

```go
// EnhancedRAGChain 實現高級 RAG 功能
type EnhancedRAGChain struct {
    name        string
    description string
    
    // 核心組件
    llm         llms.Model
    embeddings  embeddings.Embedder
    vectorStore vectorstores.VectorStore
    retriever   schema.Retriever
    
    // 文檔處理
    documentLoader *DocumentLoader
    textSplitter   textsplitter.TextSplitter
    
    // 增強功能
    contextEnricher   *ContextEnricher
    queryExpander     *QueryExpander
    resultRanker      *ResultRanker
    sourceTracker     *SourceTracker
    
    // 緩存
    cache       *cache.LRU[string, *RAGResult]
    
    // 配置
    config      *RAGConfig
    logger      *slog.Logger
}

func NewEnhancedRAGChain(config *RAGConfig, logger *slog.Logger) (*EnhancedRAGChain, error) {
    chain := &EnhancedRAGChain{
        name:        "enhanced-rag",
        description: "Enhanced Retrieval-Augmented Generation chain",
        config:      config,
        logger:      logger,
        cache:       cache.NewLRU[string, *RAGResult](config.CacheSize),
    }
    
    // 初始化 LLM
    llm, err := initializeLLM(config.LLM)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize LLM: %w", err)
    }
    chain.llm = llm
    
    // 初始化嵌入模型
    embeddings, err := initializeEmbeddings(config.Embeddings)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize embeddings: %w", err)
    }
    chain.embeddings = embeddings
    
    // 初始化向量存儲
    vectorStore, err := initializeVectorStore(config.VectorStore)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize vector store: %w", err)
    }
    chain.vectorStore = vectorStore
    
    // 創建檢索器
    retriever := vectorstores.ToRetriever(
        vectorStore,
        config.TopK,
        vectorstores.WithScoreThreshold(config.ScoreThreshold),
    )
    chain.retriever = retriever
    
    // 初始化增強組件
    chain.initializeEnhancedComponents()
    
    return chain, nil
}

func (erc *EnhancedRAGChain) ProcessQuery(ctx context.Context, query string, opts *QueryOptions) (*RAGResult, error) {
    // 檢查緩存
    if result, exists := erc.cache.Get(query); exists {
        erc.logger.Debug("Cache hit for RAG query", slog.String("query", query))
        return result, nil
    }
    
    // 1. 查詢擴展
    expandedQueries := erc.queryExpander.ExpandQuery(query, opts.Context)
    
    // 2. 多路檢索
    allDocs := make([]schema.Document, 0)
    for _, expandedQuery := range expandedQueries {
        docs, err := erc.retriever.GetRelevantDocuments(ctx, expandedQuery)
        if err != nil {
            erc.logger.Warn("Retrieval failed for expanded query", 
                slog.String("query", expandedQuery),
                slog.Any("error", err))
            continue
        }
        allDocs = append(allDocs, docs...)
    }
    
    // 3. 結果去重和排序
    uniqueDocs := erc.deduplicateDocuments(allDocs)
    rankedDocs := erc.resultRanker.RankDocuments(uniqueDocs, query, opts.Context)
    
    // 4. 上下文富化
    enrichedContext := erc.contextEnricher.EnrichContext(rankedDocs, opts.Context)
    
    // 5. 生成最終回答
    prompt := erc.buildRAGPrompt(query, enrichedContext, opts)
    response, err := erc.llm.Call(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("LLM generation failed: %w", err)
    }
    
    // 6. 結果封裝
    result := &RAGResult{
        Query:           query,
        Response:        response,
        SourceDocuments: rankedDocs,
        Sources:         erc.sourceTracker.ExtractSources(rankedDocs),
        Confidence:      erc.calculateConfidence(response, rankedDocs),
        Metadata: map[string]interface{}{
            "expanded_queries": expandedQueries,
            "total_docs":      len(allDocs),
            "unique_docs":     len(uniqueDocs),
            "final_docs":      len(rankedDocs),
        },
    }
    
    // 7. 緩存結果
    erc.cache.Set(query, result)
    
    return result, nil
}

func (erc *EnhancedRAGChain) IngestDocuments(ctx context.Context, source DocumentSource) error {
    // 1. 加載文檔
    documents, err := erc.documentLoader.LoadFromSource(ctx, source)
    if err != nil {
        return fmt.Errorf("failed to load documents: %w", err)
    }
    
    // 2. 文檔分割
    var chunks []schema.Document
    for _, doc := range documents {
        docChunks, err := erc.textSplitter.SplitDocuments([]schema.Document{doc})
        if err != nil {
            erc.logger.Warn("Failed to split document", 
                slog.String("source", doc.Metadata["source"].(string)),
                slog.Any("error", err))
            continue
        }
        chunks = append(chunks, docChunks...)
    }
    
    // 3. 生成嵌入
    embeddings, err := erc.embeddings.EmbedDocuments(ctx, 
        documentsToStrings(chunks))
    if err != nil {
        return fmt.Errorf("failed to generate embeddings: %w", err)
    }
    
    // 4. 存儲到向量存儲
    err = erc.vectorStore.AddDocuments(ctx, chunks, 
        vectorstores.WithEmbeddings(embeddings))
    if err != nil {
        return fmt.Errorf("failed to store documents: %w", err)
    }
    
    erc.logger.Info("Successfully ingested documents", 
        slog.Int("document_count", len(documents)),
        slog.Int("chunk_count", len(chunks)))
    
    return nil
}
```

### 向量存儲整合

```go
// PGVectorStore 實現 LangChain VectorStore 接口
type PGVectorStore struct {
    // Assistant 核心組件
    client    postgres.Client
    embedder  embeddings.Embedder
    
    // LangChain 兼容性
    schema.VectorStore
    
    // 配置
    tableName    string
    dimensions   int
    distanceFunc string
    
    logger *slog.Logger
}

func NewPGVectorStore(client postgres.Client, embedder embeddings.Embedder, 
                     config *VectorStoreConfig, logger *slog.Logger) *PGVectorStore {
    return &PGVectorStore{
        client:       client,
        embedder:     embedder,
        tableName:    config.TableName,
        dimensions:   config.Dimensions,
        distanceFunc: config.DistanceFunction,
        logger:       logger,
    }
}

// AddDocuments 實現 LangChain VectorStore.AddDocuments
func (pvs *PGVectorStore) AddDocuments(ctx context.Context, docs []schema.Document, 
                                      options ...vectorstores.Option) ([]string, error) {
    opts := vectorstores.NewOptions(options...)
    
    // 生成嵌入向量（如果未提供）
    var embeddings [][]float64
    if opts.Embeddings != nil {
        embeddings = opts.Embeddings
    } else {
        texts := make([]string, len(docs))
        for i, doc := range docs {
            texts[i] = doc.PageContent
        }
        
        embeds, err := pvs.embedder.EmbedDocuments(ctx, texts)
        if err != nil {
            return nil, fmt.Errorf("failed to generate embeddings: %w", err)
        }
        embeddings = embeds
    }
    
    // 轉換為 Assistant 格式
    var ids []string
    for i, doc := range docs {
        contentID := generateContentID()
        ids = append(ids, contentID)
        
        // 轉換 metadata 為 JSON
        metadataJSON, err := json.Marshal(doc.Metadata)
        if err != nil {
            pvs.logger.Warn("Failed to marshal metadata", slog.Any("error", err))
            metadataJSON = []byte("{}")
        }
        
        var metadata map[string]interface{}
        if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
            metadata = make(map[string]interface{})
        }
        
        // 存儲到 Assistant 數據庫
        _, err = pvs.client.CreateEmbedding(ctx, 
            "langchain_document",
            contentID,
            doc.PageContent,
            embeddings[i],
            metadata,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to store document %d: %w", i, err)
        }
    }
    
    return ids, nil
}

// SimilaritySearch 實現 LangChain VectorStore.SimilaritySearch
func (pvs *PGVectorStore) SimilaritySearch(ctx context.Context, query string, 
                                          numDocuments int, options ...vectorstores.Option) ([]schema.Document, error) {
    opts := vectorstores.NewOptions(options...)
    
    // 生成查詢嵌入
    queryEmbedding, err := pvs.embedder.EmbedQuery(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to embed query: %w", err)
    }
    
    // 執行相似性搜尋
    threshold := 0.0
    if opts.ScoreThreshold != nil {
        threshold = *opts.ScoreThreshold
    }
    
    results, err := pvs.client.SearchSimilarEmbeddings(ctx, 
        queryEmbedding, "langchain_document", numDocuments, threshold)
    if err != nil {
        return nil, fmt.Errorf("similarity search failed: %w", err)
    }
    
    // 轉換回 LangChain 文檔格式
    docs := make([]schema.Document, len(results))
    for i, result := range results {
        docs[i] = schema.Document{
            PageContent: result.Record.ContentText,
            Metadata:    result.Record.Metadata,
        }
        
        // 添加相似性分數
        if docs[i].Metadata == nil {
            docs[i].Metadata = make(map[string]interface{})
        }
        docs[i].Metadata["similarity_score"] = result.Similarity
        docs[i].Metadata["content_id"] = result.Record.ContentID
    }
    
    return docs, nil
}
```

## 記憶管理

### 整合記憶系統

```go
// IntegratedMemoryManager 結合 LangChain 和 Assistant 記憶
type IntegratedMemoryManager struct {
    // LangChain 記憶
    langchainMemory   schema.Memory
    
    // Assistant 記憶
    assistantMemory   *core.MemorySystem
    
    // 同步管理
    syncManager       *MemorySyncManager
    
    // 個人化管理
    personalization  *PersonalizationManager
    
    config *MemoryConfig
    logger *slog.Logger
}

func NewIntegratedMemoryManager(assistantMemory *core.MemorySystem, 
                               config *MemoryConfig, logger *slog.Logger) *IntegratedMemoryManager {
    manager := &IntegratedMemoryManager{
        assistantMemory: assistantMemory,
        config:         config,
        logger:         logger,
    }
    
    // 初始化 LangChain 記憶
    if config.LangChainMemoryType == "buffer" {
        manager.langchainMemory = memory.NewBufferMemory()
    } else if config.LangChainMemoryType == "conversation_summary" {
        manager.langchainMemory = memory.NewConversationSummaryMemory(config.LLM)
    }
    
    // 初始化同步管理器
    manager.syncManager = NewMemorySyncManager(manager.langchainMemory, 
                                              assistantMemory, logger)
    
    // 初始化個人化管理器
    manager.personalization = NewPersonalizationManager(assistantMemory, config, logger)
    
    return manager
}

func (imm *IntegratedMemoryManager) StoreInteraction(ctx context.Context, 
                                                    sessionID string, 
                                                    interaction *Interaction) error {
    // 1. 存儲到 LangChain 記憶
    err := imm.langchainMemory.SaveContext(ctx, map[string]any{
        "input":  interaction.Input,
        "output": interaction.Output,
    })
    if err != nil {
        imm.logger.Warn("Failed to store to LangChain memory", slog.Any("error", err))
    }
    
    // 2. 存儲到 Assistant 記憶
    episode := &memory.Episode{
        SessionID:   sessionID,
        UserID:      interaction.UserID,
        Interaction: interaction,
        Context:     interaction.Context,
        Timestamp:   time.Now(),
    }
    
    err = imm.assistantMemory.StoreEpisode(ctx, episode)
    if err != nil {
        imm.logger.Warn("Failed to store to Assistant memory", slog.Any("error", err))
    }
    
    // 3. 同步記憶
    go imm.syncManager.SyncMemories(ctx, sessionID)
    
    // 4. 更新個人化設定
    go imm.personalization.UpdateFromInteraction(ctx, interaction)
    
    return nil
}

func (imm *IntegratedMemoryManager) RetrieveContext(ctx context.Context, 
                                                   sessionID string, 
                                                   query string) (*MemoryContext, error) {
    // 1. 從 LangChain 記憶獲取上下文
    langchainContext, err := imm.langchainMemory.LoadMemoryVariables(ctx, 
                                                                   map[string]any{"input": query})
    if err != nil {
        imm.logger.Warn("Failed to load LangChain memory", slog.Any("error", err))
    }
    
    // 2. 從 Assistant 記憶獲取相關情節
    episodes, err := imm.assistantMemory.RetrieveRelevantEpisodes(ctx, &memory.EpisodicQuery{
        SessionID: sessionID,
        Content:   query,
        Limit:     10,
    })
    if err != nil {
        imm.logger.Warn("Failed to retrieve Assistant episodes", slog.Any("error", err))
    }
    
    // 3. 獲取個人化設定
    personalizations := imm.personalization.GetPersonalizations(ctx, sessionID)
    
    // 4. 綜合上下文
    context := &MemoryContext{
        SessionID:        sessionID,
        LangChainContext: langchainContext,
        Episodes:         episodes,
        Personalizations: personalizations,
        RetrievalTime:    time.Now(),
    }
    
    return context, nil
}
```

## 文檔處理

### 多格式文檔加載器

```go
// DocumentLoader 支援多種文檔格式的智能加載器
type DocumentLoader struct {
    // 加載器註冊表
    loaders map[string]FormatLoader
    
    // 文本分割器
    splitters map[string]textsplitter.TextSplitter
    
    // 內容識別
    contentDetector *ContentDetector
    
    // 緩存
    cache *cache.LRU[string, []schema.Document]
    
    config *DocumentLoaderConfig
    logger *slog.Logger
}

type FormatLoader interface {
    Load(ctx context.Context, source DocumentSource) ([]schema.Document, error)
    SupportedExtensions() []string
    Priority() int
}

func NewDocumentLoader(config *DocumentLoaderConfig, logger *slog.Logger) *DocumentLoader {
    dl := &DocumentLoader{
        loaders:   make(map[string]FormatLoader),
        splitters: make(map[string]textsplitter.TextSplitter),
        cache:     cache.NewLRU[string, []schema.Document](config.CacheSize),
        config:    config,
        logger:    logger,
    }
    
    // 註冊內建加載器
    dl.registerBuiltinLoaders()
    
    // 初始化文本分割器
    dl.initializeSplitters()
    
    // 初始化內容識別器
    dl.contentDetector = NewContentDetector()
    
    return dl
}

func (dl *DocumentLoader) LoadFromSource(ctx context.Context, source DocumentSource) ([]schema.Document, error) {
    // 檢查緩存
    cacheKey := dl.generateCacheKey(source)
    if docs, exists := dl.cache.Get(cacheKey); exists {
        dl.logger.Debug("Cache hit for document source", slog.String("source", source.String()))
        return docs, nil
    }
    
    var allDocs []schema.Document
    
    switch source.Type {
    case DocumentSourceTypeFile:
        docs, err := dl.loadFromFile(ctx, source.Path)
        if err != nil {
            return nil, fmt.Errorf("failed to load file: %w", err)
        }
        allDocs = docs
        
    case DocumentSourceTypeDirectory:
        docs, err := dl.loadFromDirectory(ctx, source.Path)
        if err != nil {
            return nil, fmt.Errorf("failed to load directory: %w", err)
        }
        allDocs = docs
        
    case DocumentSourceTypeURL:
        docs, err := dl.loadFromURL(ctx, source.URL)
        if err != nil {
            return nil, fmt.Errorf("failed to load URL: %w", err)
        }
        allDocs = docs
        
    case DocumentSourceTypeString:
        doc := schema.Document{
            PageContent: source.Content,
            Metadata: map[string]interface{}{
                "source": "string_input",
                "type":   "text",
            },
        }
        allDocs = []schema.Document{doc}
        
    default:
        return nil, fmt.Errorf("unsupported document source type: %v", source.Type)
    }
    
    // 緩存結果
    dl.cache.Set(cacheKey, allDocs)
    
    dl.logger.Info("Loaded documents from source", 
        slog.String("source", source.String()),
        slog.Int("document_count", len(allDocs)))
    
    return allDocs, nil
}

func (dl *DocumentLoader) loadFromFile(ctx context.Context, filePath string) ([]schema.Document, error) {
    // 檢測文件格式
    ext := filepath.Ext(filePath)
    format := dl.contentDetector.DetectFormat(filePath, ext)
    
    // 獲取對應加載器
    loader, exists := dl.loaders[format]
    if !exists {
        return nil, fmt.Errorf("no loader found for format: %s", format)
    }
    
    // 加載文檔
    source := DocumentSource{
        Type: DocumentSourceTypeFile,
        Path: filePath,
    }
    
    docs, err := loader.Load(ctx, source)
    if err != nil {
        return nil, fmt.Errorf("loader failed for %s: %w", format, err)
    }
    
    // 文本分割
    if splitter, exists := dl.splitters[format]; exists {
        splitDocs, err := splitter.SplitDocuments(docs)
        if err != nil {
            dl.logger.Warn("Text splitting failed, using original documents", 
                slog.String("format", format),
                slog.Any("error", err))
        } else {
            docs = splitDocs
        }
    }
    
    return docs, nil
}

func (dl *DocumentLoader) registerBuiltinLoaders() {
    // 文本文件加載器
    dl.loaders["txt"] = &TextFileLoader{}
    dl.loaders["md"] = &MarkdownLoader{}
    
    // 代碼文件加載器
    dl.loaders["go"] = &CodeFileLoader{Language: "go"}
    dl.loaders["py"] = &CodeFileLoader{Language: "python"}
    dl.loaders["js"] = &CodeFileLoader{Language: "javascript"}
    dl.loaders["ts"] = &CodeFileLoader{Language: "typescript"}
    
    // 配置文件加載器
    dl.loaders["yaml"] = &YAMLLoader{}
    dl.loaders["json"] = &JSONLoader{}
    
    // Web 文件加載器
    dl.loaders["html"] = &HTMLLoader{}
    
    // PDF 加載器（如果可用）
    if dl.config.EnablePDFLoader {
        dl.loaders["pdf"] = &PDFLoader{}
    }
}

func (dl *DocumentLoader) initializeSplitters() {
    // 默認文本分割器
    dl.splitters["default"] = textsplitter.NewRecursiveCharacter(
        textsplitter.WithChunkSize(dl.config.DefaultChunkSize),
        textsplitter.WithChunkOverlap(dl.config.DefaultChunkOverlap),
    )
    
    // Markdown 分割器
    dl.splitters["md"] = textsplitter.NewMarkdownTextSplitter(
        textsplitter.WithChunkSize(dl.config.MarkdownChunkSize),
        textsplitter.WithChunkOverlap(dl.config.MarkdownChunkOverlap),
    )
    
    // 代碼分割器
    codeSplitter := textsplitter.NewCodeTextSplitter(
        textsplitter.WithChunkSize(dl.config.CodeChunkSize),
        textsplitter.WithChunkOverlap(dl.config.CodeChunkOverlap),
    )
    dl.splitters["go"] = codeSplitter
    dl.splitters["py"] = codeSplitter
    dl.splitters["js"] = codeSplitter
    dl.splitters["ts"] = codeSplitter
}
```

## 配置管理

### LangChain 整合配置

```yaml
langchain:
  # 基礎配置
  enabled: true
  fallback_to_native: true
  
  # LLM 配置
  llm:
    provider: "openai"  # openai, anthropic, cohere
    model: "gpt-4"
    temperature: 0.7
    max_tokens: 2048
    
  # 嵌入模型配置
  embeddings:
    provider: "openai"
    model: "text-embedding-ada-002"
    dimensions: 1536
    
  # 向量存儲配置
  vectorstore:
    type: "pgvector"
    table_name: "langchain_embeddings"
    distance_function: "cosine"
    
  # Agent 配置
  agents:
    development:
      enabled: true
      max_iterations: 10
      early_stopping_method: "generate"
      tools:
        - "code_analyzer"
        - "git_operations"
        - "file_system"
        - "build_tools"
        
    database:
      enabled: true
      max_iterations: 8
      tools:
        - "sql_executor"
        - "schema_analyzer"
        - "query_optimizer"
        
  # RAG 配置
  rag:
    enabled: true
    top_k: 5
    score_threshold: 0.7
    cache_size: 1000
    
    # 文檔處理
    document_processing:
      chunk_size: 1000
      chunk_overlap: 200
      enable_pdf: true
      supported_formats:
        - "txt"
        - "md"
        - "go"
        - "py"
        - "js"
        - "ts"
        - "yaml"
        - "json"
        - "html"
        
  # 記憶配置
  memory:
    type: "conversation_summary"  # buffer, conversation_summary
    max_token_limit: 2000
    return_messages: true
    
    # Assistant 記憶整合
    assistant_integration:
      enabled: true
      sync_interval: "5m"
      personalization: true
```

## 性能優化

### 智能緩存策略

```go
// LangChainCacheManager 為 LangChain 操作提供智能緩存
type LangChainCacheManager struct {
    // 多層緩存
    l1Cache    *cache.LRU[string, interface{}]  // 內存緩存
    l2Cache    *cache.Redis                     // Redis 緩存
    vectorCache *cache.LRU[string, []schema.Document] // 向量緩存
    
    // 緩存策略
    strategies map[CacheType]CacheStrategy
    
    // 統計
    stats *CacheStats
    
    config *CacheConfig
    logger *slog.Logger
}

func (lcm *LangChainCacheManager) CacheLLMResponse(ctx context.Context, 
                                                  prompt string, 
                                                  response string, 
                                                  metadata map[string]interface{}) {
    // 生成緩存鍵
    key := lcm.generateLLMCacheKey(prompt, metadata)
    
    // 判斷緩存策略
    strategy := lcm.strategies[CacheTypeLLM]
    ttl := strategy.CalculateTTL(response, metadata)
    
    // L1 緩存（內存）
    lcm.l1Cache.Set(key, response)
    
    // L2 緩存（Redis）
    if lcm.l2Cache != nil {
        lcm.l2Cache.Set(ctx, key, response, ttl)
    }
    
    lcm.stats.RecordCacheSet(CacheTypeLLM)
}

func (lcm *LangChainCacheManager) GetCachedLLMResponse(ctx context.Context, 
                                                      prompt string, 
                                                      metadata map[string]interface{}) (string, bool) {
    key := lcm.generateLLMCacheKey(prompt, metadata)
    
    // 先檢查 L1 緩存
    if response, exists := lcm.l1Cache.Get(key); exists {
        lcm.stats.RecordCacheHit(CacheTypeLLM, CacheLevelL1)
        return response.(string), true
    }
    
    // 再檢查 L2 緩存
    if lcm.l2Cache != nil {
        if response, err := lcm.l2Cache.Get(ctx, key); err == nil && response != "" {
            // 回填 L1 緩存
            lcm.l1Cache.Set(key, response)
            lcm.stats.RecordCacheHit(CacheTypeLLM, CacheLevelL2)
            return response, true
        }
    }
    
    lcm.stats.RecordCacheMiss(CacheTypeLLM)
    return "", false
}

func (lcm *LangChainCacheManager) CacheVectorSearchResults(ctx context.Context, 
                                                          query string, 
                                                          results []schema.Document, 
                                                          metadata map[string]interface{}) {
    key := lcm.generateVectorCacheKey(query, metadata)
    
    // 緩存向量搜尋結果
    lcm.vectorCache.Set(key, results)
    
    lcm.stats.RecordCacheSet(CacheTypeVector)
}

func (lcm *LangChainCacheManager) GetCachedVectorSearchResults(ctx context.Context, 
                                                              query string, 
                                                              metadata map[string]interface{}) ([]schema.Document, bool) {
    key := lcm.generateVectorCacheKey(query, metadata)
    
    if results, exists := lcm.vectorCache.Get(key); exists {
        lcm.stats.RecordCacheHit(CacheTypeVector, CacheLevelL1)
        return results, true
    }
    
    lcm.stats.RecordCacheMiss(CacheTypeVector)
    return nil, false
}
```

## 監控和診斷

### LangChain 整合指標

```go
// LangChainMetrics 監控 LangChain 整合的效能指標
type LangChainMetrics struct {
    // LLM 指標
    LLMRequestCount      int64
    LLMResponseTime      time.Duration
    LLMTokenUsage        int64
    LLMErrorRate         float64
    
    // Agent 指標
    AgentExecutions      map[string]int64
    AgentSuccessRate     map[string]float64
    AgentExecutionTime   map[string]time.Duration
    
    // RAG 指標
    RAGQueryCount        int64
    RAGResponseTime      time.Duration
    RAGRetrievalAccuracy float64
    
    // 向量存儲指標
    VectorStoreSize      int64
    VectorSearchTime     time.Duration
    VectorSearchAccuracy float64
    
    // 緩存指標
    CacheHitRate         map[string]float64
    CacheMemoryUsage     int64
    
    mutex sync.RWMutex
}

func (lm *LangChainMetrics) RecordLLMRequest(duration time.Duration, tokens int64, success bool) {
    lm.mutex.Lock()
    defer lm.mutex.Unlock()
    
    lm.LLMRequestCount++
    lm.LLMResponseTime = updateRunningAverage(lm.LLMResponseTime, duration, lm.LLMRequestCount)
    lm.LLMTokenUsage += tokens
    
    if !success {
        lm.LLMErrorRate = updateErrorRate(lm.LLMErrorRate, lm.LLMRequestCount)
    }
}

func (lm *LangChainMetrics) GenerateHealthReport() *LangChainHealthReport {
    lm.mutex.RLock()
    defer lm.mutex.RUnlock()
    
    report := &LangChainHealthReport{
        Timestamp: time.Now(),
        Overall:   HealthStatusHealthy,
        Components: make(map[string]ComponentHealth),
    }
    
    // LLM 健康檢查
    llmHealth := ComponentHealth{
        Status: HealthStatusHealthy,
        Metrics: map[string]interface{}{
            "request_count":   lm.LLMRequestCount,
            "avg_response_time": lm.LLMResponseTime,
            "error_rate":      lm.LLMErrorRate,
            "token_usage":     lm.LLMTokenUsage,
        },
    }
    
    if lm.LLMErrorRate > 0.1 {
        llmHealth.Status = HealthStatusDegraded
        llmHealth.Issues = append(llmHealth.Issues, "High LLM error rate")
    }
    
    if lm.LLMResponseTime > 10*time.Second {
        llmHealth.Status = HealthStatusDegraded
        llmHealth.Issues = append(llmHealth.Issues, "Slow LLM response time")
    }
    
    report.Components["llm"] = llmHealth
    
    // Agent 健康檢查
    agentHealth := lm.assessAgentHealth()
    report.Components["agents"] = agentHealth
    
    // RAG 健康檢查
    ragHealth := lm.assessRAGHealth()
    report.Components["rag"] = ragHealth
    
    // 確定整體健康狀態
    report.Overall = lm.determineOverallHealth(llmHealth, agentHealth, ragHealth)
    
    return report
}
```

## 最佳實踐

### 1. LangChain 整合原則

- **游延加載**: 只在需要時才初始化 LangChain 組件
- **為寮倒降**: 為 LangChain 不可用時提供原生實現
- **狀態同步**: 保持 LangChain 和 Assistant 記憶的同步
- **資源管理**: 正確釋放 LangChain 資源防止洩漏

### 2. RAG 優化建議

- **分層檢索**: 實現粗精度的層次檢索策略
- **查詢擴展**: 使用同義詞和相關詞擴展查詢
- **結果重排**: 基於上下文相關性重新排序
- **緩存策略**: 合理緩存結果以提高性能

### 3. 性能調優

- **批量處理**: 合併多個請求以減少開銷
- **連接池化**: 重用 LLM 連接以提高吞吐量
- **異步處理**: 使用非同步模式處理長時間操作
- **資源監控**: 監控資源使用並及時調整

---

*LangChain 整合系統為 Assistant 提供了強大的 AI 工作流能力，通過深度整合和智能優化，實現了高效、穩定的企業級 AI 開發助手。*