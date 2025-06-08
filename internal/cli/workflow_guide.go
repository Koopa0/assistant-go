package cli

import (
	"fmt"

	"github.com/koopa0/assistant-go/internal/cli/ui"
)

// showWorkflowGuide displays workflow guidance
func (c *CLI) showWorkflowGuide() {
	fmt.Println()
	ui.Header.Println("工作流程指南")
	fmt.Println(ui.Divider())

	workflows := []struct {
		title string
		steps []string
	}{
		{
			title: "🔍 代碼審查流程",
			steps: []string{
				"1. 使用 'menu' 進入互動模式",
				"2. 選擇 'Code Analysis & Review'",
				"3. 選擇審查範圍（文件/目錄/最近更改）",
				"4. 查看分析結果並應用建議",
			},
		},
		{
			title: "🧪 測試驅動開發",
			steps: []string{
				"1. 使用 'menu' → 'Generate Tests'",
				"2. 選擇測試類型（單元/集成/基準）",
				"3. 指定要測試的函數或組件",
				"4. 審查並調整生成的測試",
			},
		},
		{
			title: "🔧 重構工作流",
			steps: []string{
				"1. 先分析代碼質量：'analyze'",
				"2. 使用 'menu' → 'Refactoring'",
				"3. 選擇重構類型",
				"4. 預覽更改並確認",
			},
		},
		{
			title: "🐛 調試流程",
			steps: []string{
				"1. 收集錯誤信息",
				"2. 使用 'menu' → 'Debug & Fix Issues'",
				"3. 選擇 'Analyze error' 貼上錯誤",
				"4. 根據建議修復問題",
			},
		},
		{
			title: "📝 文檔生成",
			steps: []string{
				"1. 確保代碼完整且可運行",
				"2. 使用 'menu' → 'Documentation'",
				"3. 選擇文檔類型",
				"4. 審查並完善生成的文檔",
			},
		},
	}

	for _, workflow := range workflows {
		ui.SubHeader.Println(workflow.title)
		for _, step := range workflow.steps {
			ui.Muted.Printf("  %s\n", step)
		}
		fmt.Println()
	}

	ui.Info.Println("💡 提示：您也可以直接輸入問題，無需進入菜單系統")
	fmt.Println()
}
