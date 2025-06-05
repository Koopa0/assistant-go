package tokens

import (
	"testing"
)

func TestTokenCounter_CountTokens(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		text      string
		expected  int32
		minTokens int32 // 最小預期 token 數
		maxTokens int32 // 最大預期 token 數
	}{
		{
			name:      "empty string",
			model:     "generic",
			text:      "",
			expected:  0,
			minTokens: 0,
			maxTokens: 0,
		},
		{
			name:      "single word",
			model:     "generic",
			text:      "hello",
			expected:  2,
			minTokens: 1,
			maxTokens: 3,
		},
		{
			name:      "english sentence",
			model:     "gpt-4",
			text:      "Hello, how are you today?",
			expected:  6,
			minTokens: 5,
			maxTokens: 8,
		},
		{
			name:      "chinese sentence",
			model:     "claude",
			text:      "你好，今天天氣怎麼樣？",
			expected:  8,
			minTokens: 6,
			maxTokens: 12,
		},
		{
			name:      "mixed chinese and english",
			model:     "claude",
			text:      "Hello 你好 world 世界",
			expected:  8,
			minTokens: 6,
			maxTokens: 12,
		},
		{
			name:      "long text",
			model:     "generic",
			text:      "This is a longer piece of text that should result in more tokens being counted by the token counter algorithm.",
			expected:  35,
			minTokens: 25,
			maxTokens: 45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTokenCounter(tt.model)
			result := tc.CountTokens(tt.text)

			// 檢查是否在合理範圍內
			if result < tt.minTokens || result > tt.maxTokens {
				t.Errorf("CountTokens() = %d, expected range [%d, %d]", result, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestTokenCounter_EstimateStructuredTokens(t *testing.T) {
	tc := NewTokenCounter("generic")

	tests := []struct {
		name      string
		content   map[string]interface{}
		minTokens int32
		maxTokens int32
	}{
		{
			name:      "empty map",
			content:   map[string]interface{}{},
			minTokens: 0,
			maxTokens: 1,
		},
		{
			name: "simple key-value",
			content: map[string]interface{}{
				"name": "Alice",
				"age":  30,
			},
			minTokens: 3,
			maxTokens: 8,
		},
		{
			name: "nested structure",
			content: map[string]interface{}{
				"user": map[string]interface{}{
					"name":  "Bob",
					"email": "bob@example.com",
				},
				"preferences": []interface{}{
					"dark_mode",
					"notifications",
				},
			},
			minTokens: 15,
			maxTokens: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tc.EstimateStructuredTokens(tt.content)

			if result < tt.minTokens || result > tt.maxTokens {
				t.Errorf("EstimateStructuredTokens() = %d, expected range [%d, %d]", result, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestTokenCounter_EstimateMessageTokens(t *testing.T) {
	tc := NewTokenCounter("generic")

	tests := []struct {
		name      string
		content   string
		metadata  map[string]interface{}
		minTokens int32
		maxTokens int32
	}{
		{
			name:      "content only",
			content:   "Hello, world!",
			metadata:  nil,
			minTokens: 3,
			maxTokens: 6,
		},
		{
			name:    "content with metadata",
			content: "Hello, world!",
			metadata: map[string]interface{}{
				"timestamp": "2023-01-01T00:00:00Z",
				"source":    "user",
			},
			minTokens: 10,
			maxTokens: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tc.EstimateMessageTokens(tt.content, tt.metadata)

			if result < tt.minTokens || result > tt.maxTokens {
				t.Errorf("EstimateMessageTokens() = %d, expected range [%d, %d]", result, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestTokenCounter_ModelSpecificCounting(t *testing.T) {
	text := "Hello 你好 world 世界"

	models := []string{"claude", "gpt-4", "generic"}
	results := make(map[string]int32)

	for _, model := range models {
		tc := NewTokenCounter(model)
		results[model] = tc.CountTokens(text)
	}

	// 確保不同模型產生合理的結果
	for model, tokens := range results {
		if tokens < 4 || tokens > 20 {
			t.Errorf("Model %s produced unreasonable token count: %d", model, tokens)
		}
	}
}

func TestTokenCounter_GetSetModel(t *testing.T) {
	tc := NewTokenCounter("claude")

	if tc.GetTokenCountingModel() != "claude" {
		t.Errorf("Expected model 'claude', got '%s'", tc.GetTokenCountingModel())
	}

	tc.SetTokenCountingModel("gpt-4")

	if tc.GetTokenCountingModel() != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", tc.GetTokenCountingModel())
	}
}

// Benchmark tests
func BenchmarkTokenCounter_CountTokens(b *testing.B) {
	tc := NewTokenCounter("generic")
	text := "This is a sample text that we'll use for benchmarking the token counting performance. It contains both English and 中文 characters to test mixed language handling."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc.CountTokens(text)
	}
}

func BenchmarkTokenCounter_EstimateStructuredTokens(b *testing.B) {
	tc := NewTokenCounter("generic")
	content := map[string]interface{}{
		"user": map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
			"preferences": map[string]interface{}{
				"theme":         "dark",
				"notifications": true,
				"language":      "zh-CN",
			},
		},
		"session": map[string]interface{}{
			"id":        "sess_123456",
			"timestamp": "2023-01-01T00:00:00Z",
			"duration":  3600,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc.EstimateStructuredTokens(content)
	}
}
