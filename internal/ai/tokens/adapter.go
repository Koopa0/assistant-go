package tokens

// ProviderAdapter 實現 AI provider 介面的適配器
type ProviderAdapter struct {
	counter Counter
}

// NewProviderAdapter 建立新的 provider adapter
func NewProviderAdapter(counter Counter) *ProviderAdapter {
	return &ProviderAdapter{
		counter: counter,
	}
}

// CountTokens 實現 AI provider 的 TokenCounter 介面
// 注意：這個方法符合 provider.go 中的 TokenCounter 介面
func (a *ProviderAdapter) CountTokens(text string, model string) (int, error) {
	// 設定模型
	originalModel := a.counter.GetTokenCountingModel()
	a.counter.SetTokenCountingModel(model)

	// 計算 tokens
	result := a.counter.CountTokens(text)

	// 恢復原始模型
	a.counter.SetTokenCountingModel(originalModel)

	return int(result), nil
}

// EstimateTokens 實現簡單的 token 估算
func (a *ProviderAdapter) EstimateTokens(text string) int {
	result := a.counter.CountTokens(text)
	return int(result)
}

// CountTokensForMessage 為完整訊息計算 tokens（包含元數據）
func (a *ProviderAdapter) CountTokensForMessage(content string, metadata map[string]interface{}) int {
	result := a.counter.EstimateMessageTokens(content, metadata)
	return int(result)
}

// CountTokensForModel 為特定模型計算 tokens
func (a *ProviderAdapter) CountTokensForModel(text string, model string) (int, error) {
	return a.CountTokens(text, model)
}
