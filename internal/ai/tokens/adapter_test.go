package tokens

import (
	"testing"
)

func TestProviderAdapter_Interface(t *testing.T) {
	// 測試 TokenCounter 是否正確實現 Counter 介面
	var _ Counter = (*TokenCounter)(nil)

	// 測試 ProviderAdapter
	counter := NewTokenCounter("generic")
	adapter := NewProviderAdapter(counter)

	text := "Hello, world!"

	// 測試 CountTokens
	count, err := adapter.CountTokens(text, "claude")
	if err != nil {
		t.Errorf("CountTokens returned error: %v", err)
	}
	if count <= 0 {
		t.Errorf("CountTokens returned invalid count: %d", count)
	}

	// 測試 EstimateTokens
	estimate := adapter.EstimateTokens(text)
	if estimate <= 0 {
		t.Errorf("EstimateTokens returned invalid estimate: %d", estimate)
	}

	// 測試 CountTokensForMessage
	metadata := map[string]interface{}{
		"timestamp": "2023-01-01T00:00:00Z",
		"source":    "user",
	}
	messageCount := adapter.CountTokensForMessage(text, metadata)
	if messageCount <= 0 {
		t.Errorf("CountTokensForMessage returned invalid count: %d", messageCount)
	}

	// 驗證 message token count 應該大於純文本 count（因為包含 metadata）
	if messageCount <= count {
		t.Errorf("Message token count (%d) should be greater than text-only count (%d)", messageCount, count)
	}
}

func TestProviderAdapter_ModelSwitching(t *testing.T) {
	counter := NewTokenCounter("generic")
	adapter := NewProviderAdapter(counter)

	text := "Hello 你好 world 世界"

	// 測試不同模型的 token 計算
	claudeCount, err := adapter.CountTokens(text, "claude")
	if err != nil {
		t.Errorf("CountTokens for claude returned error: %v", err)
	}

	gptCount, err := adapter.CountTokens(text, "gpt-4")
	if err != nil {
		t.Errorf("CountTokens for gpt-4 returned error: %v", err)
	}

	// 不同模型可能有不同的結果
	if claudeCount <= 0 || gptCount <= 0 {
		t.Errorf("Invalid token counts: claude=%d, gpt=%d", claudeCount, gptCount)
	}

	// 確保原始模型沒有被永久更改
	originalModel := counter.GetTokenCountingModel()
	if originalModel == "" {
		t.Error("Original model should not be empty")
	}
}

func TestProviderAdapter_ErrorHandling(t *testing.T) {
	counter := NewTokenCounter("generic")
	adapter := NewProviderAdapter(counter)

	// 測試空文本
	count, err := adapter.CountTokens("", "claude")
	if err != nil {
		t.Errorf("CountTokens for empty text returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("Empty text should return 0 tokens, got %d", count)
	}

	// 測試估算
	estimate := adapter.EstimateTokens("")
	if estimate != 0 {
		t.Errorf("Empty text estimate should return 0 tokens, got %d", estimate)
	}
}
