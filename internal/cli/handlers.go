package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/cli/ui"
)

// Development handlers

func (c *CLI) analyzeCodeQuality(ctx context.Context) error {
	ui.Info.Println("\n開始分析專案代碼品質...")

	// Get current directory
	cwd := "."
	ui.Info.Printf("分析目錄: %s\n", cwd)

	// Build query with context
	query := fmt.Sprintf(`請分析專案 %s 的代碼品質，包括：
1. 代碼組織和結構
2. 潛在的問題和改進點
3. 最佳實踐遵循情況
4. 測試覆蓋率建議`, cwd)

	// Use godev tool for analysis
	request := &assistant.QueryRequest{
		Query: query,
		Tools: []string{"godev"},
		Context: map[string]interface{}{
			"task_type": "code_analysis",
			"directory": cwd,
		},
	}

	stop := ui.ShowProgress("正在分析代碼...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("分析失敗: %v\n", err)
		return err
	}

	ui.Success.Println("\n✅ 代碼品質分析完成")
	fmt.Println(ui.Divider())
	fmt.Println(response.Response)

	return nil
}

func (c *CLI) generateTests(ctx context.Context) error {
	// Get target file
	targetFile := ui.InputText("請輸入要生成測試的檔案路徑", "")
	if targetFile == "" {
		ui.Warning.Println("未指定檔案")
		return nil
	}

	query := fmt.Sprintf(`為檔案 %s 生成完整的單元測試，包括：
1. 正常情況測試
2. 邊界條件測試
3. 錯誤處理測試
4. 使用 testify 框架（如果適用）`, targetFile)

	request := &assistant.QueryRequest{
		Query: query,
		Context: map[string]interface{}{
			"task_type": "test_generation",
			"file":      targetFile,
		},
	}

	stop := ui.ShowProgress("正在生成測試代碼...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("生成失敗: %v\n", err)
		return err
	}

	ui.Success.Println("\n✅ 測試代碼生成完成")
	fmt.Println(response.Response)

	// Ask if user wants to save the test
	if ui.Confirm("是否將測試保存到檔案？", true) {
		testFile := strings.TrimSuffix(targetFile, filepath.Ext(targetFile)) + "_test.go"
		ui.Info.Printf("測試將保存到: %s\n", testFile)
		// TODO: Implement file saving
	}

	return nil
}

func (c *CLI) suggestRefactoring(ctx context.Context) error {
	targetFile := ui.InputText("請輸入要重構的檔案路徑（留空分析整個專案）", "")

	var query string
	if targetFile != "" {
		query = fmt.Sprintf("分析檔案 %s 並提供詳細的重構建議", targetFile)
	} else {
		query = "分析整個專案並提供重構建議，重點關注代碼組織、設計模式和最佳實踐"
	}

	request := &assistant.QueryRequest{
		Query: query,
		Tools: []string{"godev"},
		Context: map[string]interface{}{
			"task_type": "refactoring",
		},
	}

	stop := ui.ShowProgress("正在分析重構機會...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("分析失敗: %v\n", err)
		return err
	}

	ui.Success.Println("\n✅ 重構建議已生成")
	fmt.Println(response.Response)

	return nil
}

func (c *CLI) analyzePerformance(ctx context.Context) error {
	ui.Info.Println("\n選擇性能分析類型:")
	ui.Muted.Println("1. CPU 性能分析")
	ui.Muted.Println("2. 記憶體使用分析")
	ui.Muted.Println("3. 並發性能分析")
	ui.Muted.Println("4. 整體性能審查")

	fmt.Print("請選擇 (1-4): ")
	var choice string
	fmt.Scanln(&choice)

	analysisType := map[string]string{
		"1": "CPU 使用和熱點",
		"2": "記憶體分配和洩漏",
		"3": "goroutine 和並發模式",
		"4": "整體性能",
	}[choice]

	if analysisType == "" {
		ui.Warning.Println("無效的選擇")
		return nil
	}

	query := fmt.Sprintf("分析專案的%s，提供優化建議", analysisType)

	request := &assistant.QueryRequest{
		Query: query,
		Tools: []string{"godev"},
		Context: map[string]interface{}{
			"task_type":     "performance",
			"analysis_type": analysisType,
		},
	}

	stop := ui.ShowProgress("正在進行性能分析...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("分析失敗: %v\n", err)
		return err
	}

	ui.Success.Printf("\n✅ %s分析完成\n", analysisType)
	fmt.Println(response.Response)

	return nil
}

// Database handlers

func (c *CLI) optimizeSQL(ctx context.Context) error {
	ui.Info.Println("\n請輸入要優化的 SQL 查詢（輸入 'done' 結束）:")

	var sqlQuery strings.Builder
	for {
		var line string
		fmt.Scanln(&line)
		if line == "done" {
			break
		}
		sqlQuery.WriteString(line + "\n")
	}

	if sqlQuery.Len() == 0 {
		ui.Warning.Println("未輸入查詢")
		return nil
	}

	query := fmt.Sprintf("優化以下 SQL 查詢並解釋優化原理：\n%s", sqlQuery.String())

	request := &assistant.QueryRequest{
		Query: query,
		Tools: []string{"postgres"},
		Context: map[string]interface{}{
			"task_type": "query_optimization",
		},
	}

	stop := ui.ShowProgress("正在優化查詢...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("優化失敗: %v\n", err)
		return err
	}

	ui.Success.Println("\n✅ SQL 優化完成")
	fmt.Println(response.Response)

	return nil
}

func (c *CLI) generateMigration(ctx context.Context) error {
	ui.Info.Println("\n生成資料庫遷移")

	migrationType := ui.SelectOption("選擇遷移類型", []string{
		"添加新表",
		"修改現有表",
		"添加索引",
		"修改欄位",
		"自定義遷移",
	})

	description := ui.InputText("請描述遷移的具體需求", "")

	query := fmt.Sprintf("生成 PostgreSQL 遷移檔案：%s - %s",
		[]string{"添加新表", "修改現有表", "添加索引", "修改欄位", "自定義遷移"}[migrationType],
		description)

	request := &assistant.QueryRequest{
		Query: query,
		Tools: []string{"postgres"},
		Context: map[string]interface{}{
			"task_type":      "migration_generation",
			"migration_type": migrationType,
		},
	}

	stop := ui.ShowProgress("正在生成遷移...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("生成失敗: %v\n", err)
		return err
	}

	ui.Success.Println("\n✅ 遷移檔案已生成")
	fmt.Println(response.Response)

	return nil
}

func (c *CLI) analyzeSchema(ctx context.Context) error {
	tableName := ui.InputText("請輸入要分析的表名（留空分析整個架構）", "")

	var query string
	if tableName != "" {
		query = fmt.Sprintf("分析表 %s 的架構並提供索引優化建議", tableName)
	} else {
		query = "分析整個資料庫架構，識別性能瓶頸並提供優化建議"
	}

	request := &assistant.QueryRequest{
		Query: query,
		Tools: []string{"postgres"},
		Context: map[string]interface{}{
			"task_type": "schema_analysis",
		},
	}

	stop := ui.ShowProgress("正在分析架構...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("分析失敗: %v\n", err)
		return err
	}

	ui.Success.Println("\n✅ 架構分析完成")
	fmt.Println(response.Response)

	return nil
}

// DevOps handlers

func (c *CLI) analyzeDockerfile(ctx context.Context) error {
	dockerfilePath := ui.InputText("請輸入 Dockerfile 路徑", "./Dockerfile")

	query := fmt.Sprintf("分析 %s 並提供優化建議，重點關注：\n1. 映像大小優化\n2. 構建速度\n3. 安全性\n4. 最佳實踐", dockerfilePath)

	request := &assistant.QueryRequest{
		Query: query,
		Tools: []string{"docker"},
		Context: map[string]interface{}{
			"task_type": "dockerfile_analysis",
			"file":      dockerfilePath,
		},
	}

	stop := ui.ShowProgress("正在分析 Dockerfile...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("分析失敗: %v\n", err)
		return err
	}

	ui.Success.Println("\n✅ Dockerfile 分析完成")
	fmt.Println(response.Response)

	return nil
}

func (c *CLI) checkK8sConfig(ctx context.Context) error {
	ui.Warning.Println("Kubernetes 功能尚在開發中...")

	// Placeholder for K8s functionality
	query := "檢查 Kubernetes 配置的最佳實踐"

	request := &assistant.QueryRequest{
		Query: query,
		Context: map[string]interface{}{
			"task_type": "k8s_config_check",
		},
	}

	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	if err != nil {
		ui.Error.Printf("檢查失敗: %v\n", err)
		return err
	}

	fmt.Println(response.Response)
	return nil
}

func (c *CLI) optimizeCICD(ctx context.Context) error {
	ciPlatform := ui.SelectOption("選擇 CI/CD 平台", []string{
		"GitHub Actions",
		"GitLab CI",
		"Jenkins",
		"CircleCI",
		"其他",
	})

	platformName := []string{
		"GitHub Actions",
		"GitLab CI",
		"Jenkins",
		"CircleCI",
		"其他",
	}[ciPlatform]

	query := fmt.Sprintf("分析並優化 %s 的 CI/CD 管道配置，提供最佳實踐建議", platformName)

	request := &assistant.QueryRequest{
		Query: query,
		Context: map[string]interface{}{
			"task_type": "cicd_optimization",
			"platform":  platformName,
		},
	}

	stop := ui.ShowProgress("正在分析 CI/CD 配置...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("分析失敗: %v\n", err)
		return err
	}

	ui.Success.Println("\n✅ CI/CD 優化建議已生成")
	fmt.Println(response.Response)

	return nil
}

// Project analysis handlers

func (c *CLI) scanProjectStructure(ctx context.Context) error {
	ui.Info.Println("\n掃描專案結構...")

	query := "深入分析專案結構，包括目錄組織、模組劃分、依賴關係和架構模式"

	request := &assistant.QueryRequest{
		Query: query,
		Tools: []string{"godev"},
		Context: map[string]interface{}{
			"task_type": "workspace_analysis",
		},
	}

	stop := ui.ShowProgress("正在掃描專案...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("掃描失敗: %v\n", err)
		return err
	}

	ui.Success.Println("\n✅ 專案結構掃描完成")
	fmt.Println(response.Response)

	return nil
}

func (c *CLI) analyzeDependencies(ctx context.Context) error {
	query := "分析專案的依賴關係，包括直接和間接依賴，識別潛在的安全問題和版本衝突"

	request := &assistant.QueryRequest{
		Query: query,
		Tools: []string{"godev"},
		Context: map[string]interface{}{
			"task_type": "dependency_analysis",
		},
	}

	stop := ui.ShowProgress("正在分析依賴...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("分析失敗: %v\n", err)
		return err
	}

	ui.Success.Println("\n✅ 依賴分析完成")
	fmt.Println(response.Response)

	return nil
}

func (c *CLI) architectureReview(ctx context.Context) error {
	query := "進行全面的架構審查，評估系統設計、模組耦合度、擴展性和維護性"

	request := &assistant.QueryRequest{
		Query: query,
		Tools: []string{"godev"},
		Context: map[string]interface{}{
			"task_type": "architecture_review",
		},
	}

	stop := ui.ShowProgress("正在進行架構審查...")
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
	stop()

	if err != nil {
		ui.Error.Printf("審查失敗: %v\n", err)
		return err
	}

	ui.Success.Println("\n✅ 架構審查完成")
	fmt.Println(response.Response)

	return nil
}

// LangChain handlers

func (c *CLI) runDevelopmentAgent(ctx context.Context) error {
	task := ui.InputText("請描述開發任務", "")
	if task == "" {
		ui.Warning.Println("未輸入任務")
		return nil
	}

	c.executeLangChainAgent(ctx, "development", task)
	return nil
}

func (c *CLI) runDatabaseAgent(ctx context.Context) error {
	task := ui.InputText("請描述資料庫任務", "")
	if task == "" {
		ui.Warning.Println("未輸入任務")
		return nil
	}

	c.executeLangChainAgent(ctx, "database", task)
	return nil
}

func (c *CLI) runRAGQuery(ctx context.Context) error {
	ui.Info.Println("\nRAG (Retrieval-Augmented Generation) 查詢")

	docSource := ui.SelectOption("選擇文檔來源", []string{
		"本地檔案",
		"專案文檔",
		"知識庫",
		"URL",
	})

	var source string
	switch docSource {
	case 0:
		source = ui.InputText("請輸入檔案路徑", "")
	case 1:
		source = "./docs"
	case 2:
		source = "knowledge_base"
	case 3:
		source = ui.InputText("請輸入 URL", "")
	}

	query := ui.InputText("請輸入查詢問題", "")

	// Execute RAG chain
	ui.Info.Printf("正在從 %s 檢索相關資訊...\n", source)
	ui.Info.Printf("查詢: %s\n", query)
	ui.Warning.Println("RAG 功能正在開發中")

	return nil
}

func (c *CLI) manageMemory(ctx context.Context) error {
	option := ui.SelectOption("記憶管理選項", []string{
		"查看當前記憶",
		"清除工作記憶",
		"匯出記憶",
		"匯入記憶",
	})

	switch option {
	case 0:
		ui.Info.Println("查看記憶系統...")
		// TODO: Implement memory viewing
	case 1:
		if ui.Confirm("確定要清除工作記憶嗎？", false) {
			ui.Success.Println("工作記憶已清除")
		}
	case 2:
		ui.Info.Println("匯出記憶...")
		// TODO: Implement memory export
	case 3:
		ui.Info.Println("匯入記憶...")
		// TODO: Implement memory import
	}

	return nil
}
