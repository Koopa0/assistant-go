package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/koopa0/assistant-go/internal/cli/ui"
)

// InteractiveMenu represents a menu item
type InteractiveMenu struct {
	Title   string
	Options []MenuOption
}

// MenuOption represents a single menu option
type MenuOption struct {
	Key         string
	Description string
	Handler     func(ctx context.Context, cli *CLI) error
}

// showMainMenu displays the main interactive menu
func (c *CLI) showMainMenu(ctx context.Context) error {
	menu := InteractiveMenu{
		Title: "選擇您要執行的任務類型",
		Options: []MenuOption{
			{
				Key:         "1",
				Description: "💻 開發輔助 - 代碼分析、重構建議、測試生成",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.developmentMenu(ctx)
				},
			},
			{
				Key:         "2",
				Description: "🗄️ 資料庫操作 - SQL 優化、架構分析、遷移生成",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.databaseMenu(ctx)
				},
			},
			{
				Key:         "3",
				Description: "🐳 DevOps 工具 - Docker、Kubernetes、CI/CD",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.devopsMenu(ctx)
				},
			},
			{
				Key:         "4",
				Description: "📊 專案分析 - 工作區掃描、依賴分析、架構審查",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.projectAnalysisMenu(ctx)
				},
			},
			{
				Key:         "5",
				Description: "🤖 LangChain 功能 - 智能代理、記憶系統、RAG",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.langchainMenu(ctx)
				},
			},
			{
				Key:         "6",
				Description: "💬 自由對話 - 開放式問答和討論",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.freeConversation(ctx)
				},
			},
			{
				Key:         "0",
				Description: "返回主選單",
				Handler:     nil,
			},
		},
	}

	return c.displayMenu(ctx, menu)
}

// displayMenu shows a menu and handles user selection
func (c *CLI) displayMenu(ctx context.Context, menu InteractiveMenu) error {
	ui.Header.Println("\n" + menu.Title)
	fmt.Println(ui.Divider())

	for _, option := range menu.Options {
		ui.Label.Printf("  [%s] ", option.Key)
		ui.Info.Println(option.Description)
	}

	fmt.Print("\n請選擇: ")

	var choice string
	fmt.Scanln(&choice)
	choice = strings.TrimSpace(choice)

	for _, option := range menu.Options {
		if option.Key == choice {
			if option.Handler != nil {
				return option.Handler(ctx, c)
			}
			return nil
		}
	}

	ui.Warning.Println("無效的選擇，請重試")
	return c.displayMenu(ctx, menu)
}

// developmentMenu shows development-related options
func (c *CLI) developmentMenu(ctx context.Context) error {
	menu := InteractiveMenu{
		Title: "開發輔助功能",
		Options: []MenuOption{
			{
				Key:         "1",
				Description: "分析當前專案代碼品質",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.analyzeCodeQuality(ctx)
				},
			},
			{
				Key:         "2",
				Description: "生成單元測試",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.generateTests(ctx)
				},
			},
			{
				Key:         "3",
				Description: "重構建議",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.suggestRefactoring(ctx)
				},
			},
			{
				Key:         "4",
				Description: "性能分析",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.analyzePerformance(ctx)
				},
			},
			{
				Key:         "0",
				Description: "返回上級選單",
				Handler:     nil,
			},
		},
	}

	return c.displayMenu(ctx, menu)
}

// databaseMenu shows database-related options
func (c *CLI) databaseMenu(ctx context.Context) error {
	menu := InteractiveMenu{
		Title: "資料庫操作",
		Options: []MenuOption{
			{
				Key:         "1",
				Description: "SQL 查詢優化",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.optimizeSQL(ctx)
				},
			},
			{
				Key:         "2",
				Description: "生成資料庫遷移",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.generateMigration(ctx)
				},
			},
			{
				Key:         "3",
				Description: "架構分析與索引建議",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.analyzeSchema(ctx)
				},
			},
			{
				Key:         "0",
				Description: "返回上級選單",
				Handler:     nil,
			},
		},
	}

	return c.displayMenu(ctx, menu)
}

// devopsMenu shows DevOps-related options
func (c *CLI) devopsMenu(ctx context.Context) error {
	menu := InteractiveMenu{
		Title: "DevOps 工具",
		Options: []MenuOption{
			{
				Key:         "1",
				Description: "分析並優化 Dockerfile",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.analyzeDockerfile(ctx)
				},
			},
			{
				Key:         "2",
				Description: "Kubernetes 資源配置檢查",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.checkK8sConfig(ctx)
				},
			},
			{
				Key:         "3",
				Description: "CI/CD 管道優化建議",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.optimizeCICD(ctx)
				},
			},
			{
				Key:         "0",
				Description: "返回上級選單",
				Handler:     nil,
			},
		},
	}

	return c.displayMenu(ctx, menu)
}

// projectAnalysisMenu shows project analysis options
func (c *CLI) projectAnalysisMenu(ctx context.Context) error {
	menu := InteractiveMenu{
		Title: "專案分析",
		Options: []MenuOption{
			{
				Key:         "1",
				Description: "掃描專案結構",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.scanProjectStructure(ctx)
				},
			},
			{
				Key:         "2",
				Description: "分析依賴關係",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.analyzeDependencies(ctx)
				},
			},
			{
				Key:         "3",
				Description: "架構審查報告",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.architectureReview(ctx)
				},
			},
			{
				Key:         "0",
				Description: "返回上級選單",
				Handler:     nil,
			},
		},
	}

	return c.displayMenu(ctx, menu)
}

// langchainMenu shows LangChain-related options
func (c *CLI) langchainMenu(ctx context.Context) error {
	menu := InteractiveMenu{
		Title: "LangChain 功能",
		Options: []MenuOption{
			{
				Key:         "1",
				Description: "執行開發代理",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.runDevelopmentAgent(ctx)
				},
			},
			{
				Key:         "2",
				Description: "執行資料庫代理",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.runDatabaseAgent(ctx)
				},
			},
			{
				Key:         "3",
				Description: "執行 RAG 查詢",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.runRAGQuery(ctx)
				},
			},
			{
				Key:         "4",
				Description: "管理記憶系統",
				Handler: func(ctx context.Context, cli *CLI) error {
					return cli.manageMemory(ctx)
				},
			},
			{
				Key:         "0",
				Description: "返回上級選單",
				Handler:     nil,
			},
		},
	}

	return c.displayMenu(ctx, menu)
}

// freeConversation starts a free conversation mode
func (c *CLI) freeConversation(ctx context.Context) error {
	ui.Info.Println("\n進入自由對話模式。輸入 'menu' 返回主選單。")
	fmt.Println(ui.Divider())

	for {
		fmt.Print("\n💬 > ")
		var input string
		fmt.Scanln(&input)

		if strings.ToLower(input) == "menu" {
			return nil
		}

		// Process the query
		c.processQuery(ctx, input)
	}
}

// Workflow guidance functions
func (c *CLI) showWorkflowGuide() {
	guides := []struct {
		workflow string
		steps    []string
	}{
		{
			workflow: "🚀 快速開始工作流程",
			steps: []string{
				"1. 使用 'status' 檢查系統狀態",
				"2. 使用 'tools' 查看可用工具",
				"3. 選擇合適的任務類型開始工作",
				"4. 跟隨引導完成任務",
			},
		},
		{
			workflow: "🔍 代碼審查工作流程",
			steps: []string{
				"1. 選擇 '開發輔助' > '分析當前專案代碼品質'",
				"2. 系統會掃描專案並生成報告",
				"3. 根據報告選擇 '重構建議' 獲取改進方案",
				"4. 使用 '生成單元測試' 提高測試覆蓋率",
			},
		},
		{
			workflow: "🗄️ 資料庫優化工作流程",
			steps: []string{
				"1. 選擇 '資料庫操作' > 'SQL 查詢優化'",
				"2. 輸入需要優化的查詢",
				"3. 查看優化建議和執行計劃",
				"4. 使用 '架構分析與索引建議' 進一步優化",
			},
		},
	}

	ui.Header.Println("\n工作流程指南")
	for _, guide := range guides {
		ui.SubHeader.Println("\n" + guide.workflow)
		for _, step := range guide.steps {
			ui.Muted.Println("  " + step)
		}
	}
}

// Context persistence helpers
func (c *CLI) saveContext(key string, value interface{}) error {
	// Save context to memory or database
	// This helps maintain conversation continuity
	return nil
}

func (c *CLI) loadContext(key string) (interface{}, error) {
	// Load previously saved context
	return nil, nil
}
