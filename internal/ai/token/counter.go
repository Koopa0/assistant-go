package token

import (
	"fmt"
	"regexp"
	"strings"
)

// Counter 定義 token 計算的介面
type Counter interface {
	// CountTokens 計算文本的 token 數量
	CountTokens(text string) int32

	// EstimateStructuredTokens 估算結構化內容的 token 數量
	EstimateStructuredTokens(content map[string]interface{}) int32

	// EstimateMessageTokens 估算訊息的總 token 數量（包含內容和元數據）
	EstimateMessageTokens(content string, metadata map[string]interface{}) int32

	// GetTokenCountingModel 取得當前 token 計算所使用的模型
	GetTokenCountingModel() string

	// SetTokenCountingModel 設定 token 計算所使用的模型
	SetTokenCountingModel(model string)
}

// TokenCounter 提供 token 計算功能
type TokenCounter struct {
	// 可以根據不同模型調整計算策略
	model string
}

// NewTokenCounter 建立新的 token 計算器
func NewTokenCounter(model string) *TokenCounter {
	return &TokenCounter{
		model: model,
	}
}

// CountTokens 計算文本的 token 數量
// 使用近似估算：英文約 4 字符 = 1 token，中文約 1.5 字符 = 1 token
func (tc *TokenCounter) CountTokens(text string) int32 {
	if text == "" {
		return 0
	}

	// 移除多餘的空白字符
	cleanText := strings.TrimSpace(text)
	if cleanText == "" {
		return 0
	}

	// 基於模型的估算策略
	switch tc.model {
	case "claude", "claude-3", "claude-3.5":
		return tc.countClaude(cleanText)
	case "gpt-3.5", "gpt-4", "gpt-4-turbo":
		return tc.countGPT(cleanText)
	default:
		return tc.countGeneric(cleanText)
	}
}

// countClaude Claude 模型的 token 估算
func (tc *TokenCounter) countClaude(text string) int32 {
	// Claude 對中英文的處理相對均衡
	charCount := len([]rune(text))
	chineseCount := tc.countChinese(text)
	englishCount := charCount - chineseCount

	// 中文: 約 1.5 字符 = 1 token
	// 英文: 約 4 字符 = 1 token
	chineseTokens := float64(chineseCount) / 1.5
	englishTokens := float64(englishCount) / 4.0

	total := chineseTokens + englishTokens

	// 最少 1 token
	if total < 1 {
		return 1
	}

	return int32(total + 0.5) // 四捨五入
}

// countGPT GPT 模型的 token 估算
func (tc *TokenCounter) countGPT(text string) int32 {
	// GPT 模型對英文較優化，中文 token 消耗較高
	charCount := len([]rune(text))
	chineseCount := tc.countChinese(text)
	englishCount := charCount - chineseCount

	// 中文: 約 1 字符 = 1 token (較保守估算)
	// 英文: 約 4 字符 = 1 token
	chineseTokens := float64(chineseCount)
	englishTokens := float64(englishCount) / 4.0

	total := chineseTokens + englishTokens

	// 最少 1 token
	if total < 1 {
		return 1
	}

	return int32(total + 0.5) // 四捨五入
}

// countGeneric 通用 token 估算
func (tc *TokenCounter) countGeneric(text string) int32 {
	// 使用保守的估算策略
	charCount := len([]rune(text))

	// 保守估算: 平均 3 字符 = 1 token
	tokens := float64(charCount) / 3.0

	// 最少 1 token
	if tokens < 1 {
		return 1
	}

	return int32(tokens + 0.5) // 四捨五入
}

// countChinese 計算中文字符數量
func (tc *TokenCounter) countChinese(text string) int {
	// 中文字符範圍的正則表達式
	chineseRegex := regexp.MustCompile(`[\p{Han}]`)
	matches := chineseRegex.FindAllString(text, -1)
	return len(matches)
}

// EstimateStructuredTokens 估算結構化內容的 token 數量
func (tc *TokenCounter) EstimateStructuredTokens(content map[string]interface{}) int32 {
	total := int32(0)

	for key, value := range content {
		// 計算鍵的 token
		total += tc.CountTokens(key)

		// 計算值的 token
		switch v := value.(type) {
		case string:
			total += tc.CountTokens(v)
		case map[string]interface{}:
			total += tc.EstimateStructuredTokens(v)
		case []interface{}:
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					total += tc.EstimateStructuredTokens(itemMap)
				} else if itemStr, ok := item.(string); ok {
					total += tc.CountTokens(itemStr)
				} else {
					total += tc.CountTokens(fmt.Sprintf("%v", item))
				}
			}
		default:
			// 其他類型轉換為字符串計算
			total += tc.CountTokens(fmt.Sprintf("%v", v))
		}
	}

	// 結構化內容的額外開銷 (JSON 括號、逗號等)
	overhead := int32(float64(total) * 0.1) // 10% 開銷
	return total + overhead
}

// EstimateMessageTokens 估算訊息的總 token 數量（包含內容和元數據）
func (tc *TokenCounter) EstimateMessageTokens(content string, metadata map[string]interface{}) int32 {
	contentTokens := tc.CountTokens(content)

	var metadataTokens int32
	if metadata != nil && len(metadata) > 0 {
		metadataTokens = tc.EstimateStructuredTokens(metadata)
	}

	return contentTokens + metadataTokens
}

// GetTokenCountingModel 取得當前 token 計算所使用的模型
func (tc TokenCounter) GetTokenCountingModel() string {
	return tc.model
}

// SetTokenCountingModel 設定 token 計算所使用的模型
func (tc *TokenCounter) SetTokenCountingModel(model string) {
	tc.model = model
}
