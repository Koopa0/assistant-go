package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

// Formatter 提供內容格式化功能
type Formatter struct {
	colorizer *Colorizer
}

// NewFormatter 建立新的格式化器
func NewFormatter() *Formatter {
	return &Formatter{
		colorizer: NewColorizer("auto"),
	}
}

// FormatContent 格式化內容，支援 Markdown 和語法高亮
func (f *Formatter) FormatContent(content string, syntaxHighlight bool) string {
	if !syntaxHighlight {
		return content
	}

	// 處理各種 Markdown 元素
	content = f.formatHeaders(content)
	content = f.formatCodeBlocks(content)
	content = f.formatInlineCode(content)
	content = f.formatLinks(content)
	content = f.formatBold(content)
	content = f.formatItalic(content)
	content = f.formatQuotes(content)
	content = f.formatLists(content)

	return content
}

// formatHeaders 格式化標題
func (f *Formatter) formatHeaders(content string) string {
	// H1: # 標題
	h1Pattern := regexp.MustCompile(`^# (.+)$`)
	content = h1Pattern.ReplaceAllStringFunc(content, func(match string) string {
		title := h1Pattern.FindStringSubmatch(match)[1]
		return f.colorizer.Title(fmt.Sprintf("# %s", title))
	})

	// H2: ## 標題
	h2Pattern := regexp.MustCompile(`^## (.+)$`)
	content = h2Pattern.ReplaceAllStringFunc(content, func(match string) string {
		title := h2Pattern.FindStringSubmatch(match)[1]
		return f.colorizer.Subtitle(fmt.Sprintf("## %s", title))
	})

	// H3: ### 標題
	h3Pattern := regexp.MustCompile(`^### (.+)$`)
	content = h3Pattern.ReplaceAllStringFunc(content, func(match string) string {
		title := h3Pattern.FindStringSubmatch(match)[1]
		return f.colorizer.Info(fmt.Sprintf("### %s", title))
	})

	return content
}

// formatCodeBlocks 格式化程式碼區塊
func (f *Formatter) formatCodeBlocks(content string) string {
	// 三重反引號程式碼區塊
	codeBlockPattern := regexp.MustCompile("(?s)```(\\w+)?\\n?(.*?)```")
	content = codeBlockPattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := codeBlockPattern.FindStringSubmatch(match)
		language := parts[1]
		code := parts[2]

		// 添加程式碼區塊樣式
		formattedCode := f.formatCode(code, language)

		// 添加邊框
		lines := strings.Split(formattedCode, "\n")
		var result strings.Builder

		result.WriteString(f.colorizer.Dim("┌─────────────────────────────────────────────────────────────────\n"))
		if language != "" {
			result.WriteString(f.colorizer.Dim(fmt.Sprintf("│ %s\n", language)))
			result.WriteString(f.colorizer.Dim("├─────────────────────────────────────────────────────────────────\n"))
		}

		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				result.WriteString(f.colorizer.Dim("│ "))
				result.WriteString(line)
				result.WriteString("\n")
			}
		}

		result.WriteString(f.colorizer.Dim("└─────────────────────────────────────────────────────────────────"))

		return result.String()
	})

	return content
}

// formatCode 格式化程式碼並添加語法高亮
func (f *Formatter) formatCode(code, language string) string {
	switch strings.ToLower(language) {
	case "go", "golang":
		return f.formatGoCode(code)
	case "javascript", "js":
		return f.formatJSCode(code)
	case "python", "py":
		return f.formatPythonCode(code)
	case "json":
		return f.formatJSONCode(code)
	case "yaml", "yml":
		return f.formatYAMLCode(code)
	case "sql":
		return f.formatSQLCode(code)
	default:
		return f.colorizer.Code(code)
	}
}

// formatGoCode 格式化 Go 程式碼
func (f *Formatter) formatGoCode(code string) string {
	// Go 關鍵字
	keywords := []string{
		"package", "import", "func", "var", "const", "type", "struct", "interface",
		"if", "else", "for", "range", "switch", "case", "default", "select",
		"go", "defer", "return", "break", "continue", "fallthrough",
		"chan", "map", "slice", "string", "int", "bool", "error",
	}

	result := code

	// 高亮關鍵字
	for _, keyword := range keywords {
		pattern := regexp.MustCompile(`\b` + keyword + `\b`)
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			return color.New(color.FgBlue, color.Bold).Sprint(match)
		})
	}

	// 高亮字串
	stringPattern := regexp.MustCompile(`"([^"]*)"`)
	result = stringPattern.ReplaceAllStringFunc(result, func(match string) string {
		return color.New(color.FgGreen).Sprint(match)
	})

	// 高亮註解
	commentPattern := regexp.MustCompile(`//.*$`)
	result = commentPattern.ReplaceAllStringFunc(result, func(match string) string {
		return color.New(color.Faint).Sprint(match)
	})

	return result
}

// formatJSCode 格式化 JavaScript 程式碼
func (f *Formatter) formatJSCode(code string) string {
	keywords := []string{
		"function", "var", "let", "const", "if", "else", "for", "while",
		"do", "switch", "case", "default", "break", "continue", "return",
		"try", "catch", "finally", "throw", "class", "extends", "import", "export",
	}

	result := code

	for _, keyword := range keywords {
		pattern := regexp.MustCompile(`\b` + keyword + `\b`)
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			return color.New(color.FgBlue, color.Bold).Sprint(match)
		})
	}

	return f.highlightStringsAndComments(result)
}

// formatPythonCode 格式化 Python 程式碼
func (f *Formatter) formatPythonCode(code string) string {
	keywords := []string{
		"def", "class", "if", "elif", "else", "for", "while", "try", "except",
		"finally", "with", "as", "import", "from", "return", "break", "continue",
		"pass", "lambda", "yield", "global", "nonlocal", "assert", "del",
	}

	result := code

	for _, keyword := range keywords {
		pattern := regexp.MustCompile(`\b` + keyword + `\b`)
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			return color.New(color.FgBlue, color.Bold).Sprint(match)
		})
	}

	// Python 註解使用 #
	commentPattern := regexp.MustCompile(`#.*$`)
	result = commentPattern.ReplaceAllStringFunc(result, func(match string) string {
		return color.New(color.Faint).Sprint(match)
	})

	return f.highlightStringsAndComments(result)
}

// formatJSONCode 格式化 JSON
func (f *Formatter) formatJSONCode(code string) string {
	result := code

	// 高亮鍵
	keyPattern := regexp.MustCompile(`"([^"]+)":`)
	result = keyPattern.ReplaceAllStringFunc(result, func(match string) string {
		return color.New(color.FgBlue).Sprint(match)
	})

	// 高亮字串值
	stringPattern := regexp.MustCompile(`: "([^"]*)"`)
	result = stringPattern.ReplaceAllStringFunc(result, func(match string) string {
		return color.New(color.FgGreen).Sprint(match)
	})

	// 高亮數字
	numberPattern := regexp.MustCompile(`: (-?\d+\.?\d*)`)
	result = numberPattern.ReplaceAllStringFunc(result, func(match string) string {
		return color.New(color.FgYellow).Sprint(match)
	})

	// 高亮布林值和 null
	boolNullPattern := regexp.MustCompile(`\b(true|false|null)\b`)
	result = boolNullPattern.ReplaceAllStringFunc(result, func(match string) string {
		return color.New(color.FgMagenta).Sprint(match)
	})

	return result
}

// formatYAMLCode 格式化 YAML
func (f *Formatter) formatYAMLCode(code string) string {
	result := code

	// 高亮鍵
	keyPattern := regexp.MustCompile(`^(\s*)([^:\s]+):`)
	result = keyPattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := keyPattern.FindStringSubmatch(match)
		indent := parts[1]
		key := parts[2]
		return indent + color.New(color.FgBlue).Sprint(key) + ":"
	})

	// 高亮字串值
	stringPattern := regexp.MustCompile(`: (.+)$`)
	result = stringPattern.ReplaceAllStringFunc(result, func(match string) string {
		value := strings.TrimPrefix(match, ": ")
		return ": " + color.New(color.FgGreen).Sprint(value)
	})

	return result
}

// formatSQLCode 格式化 SQL
func (f *Formatter) formatSQLCode(code string) string {
	keywords := []string{
		"SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER",
		"GROUP", "ORDER", "BY", "HAVING", "INSERT", "UPDATE", "DELETE",
		"CREATE", "ALTER", "DROP", "TABLE", "INDEX", "VIEW", "AND", "OR", "NOT",
	}

	result := code

	for _, keyword := range keywords {
		pattern := regexp.MustCompile(`(?i)\b` + keyword + `\b`)
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			return color.New(color.FgBlue, color.Bold).Sprint(strings.ToUpper(match))
		})
	}

	return f.highlightStringsAndComments(result)
}

// highlightStringsAndComments 高亮字串和註解
func (f *Formatter) highlightStringsAndComments(code string) string {
	result := code

	// 高亮字串
	stringPattern := regexp.MustCompile(`"([^"]*)"`)
	result = stringPattern.ReplaceAllStringFunc(result, func(match string) string {
		return color.New(color.FgGreen).Sprint(match)
	})

	// 高亮單引號字串
	singleStringPattern := regexp.MustCompile(`'([^']*)'`)
	result = singleStringPattern.ReplaceAllStringFunc(result, func(match string) string {
		return color.New(color.FgGreen).Sprint(match)
	})

	return result
}

// formatInlineCode 格式化行內程式碼
func (f *Formatter) formatInlineCode(content string) string {
	inlineCodePattern := regexp.MustCompile("`([^`]+)`")
	return inlineCodePattern.ReplaceAllStringFunc(content, func(match string) string {
		code := strings.Trim(match, "`")
		return f.colorizer.Code(fmt.Sprintf("`%s`", code))
	})
}

// formatLinks 格式化連結
func (f *Formatter) formatLinks(content string) string {
	// Markdown 連結格式: [文字](URL)
	linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	return linkPattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := linkPattern.FindStringSubmatch(match)
		text := parts[1]
		url := parts[2]
		return f.colorizer.Link(text) + f.colorizer.Dim(fmt.Sprintf(" (%s)", url))
	})
}

// formatBold 格式化粗體
func (f *Formatter) formatBold(content string) string {
	boldPattern := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	return boldPattern.ReplaceAllStringFunc(content, func(match string) string {
		text := strings.Trim(match, "*")
		return f.colorizer.Bold(text)
	})
}

// formatItalic 格式化斜體
func (f *Formatter) formatItalic(content string) string {
	italicPattern := regexp.MustCompile(`\*([^*]+)\*`)
	return italicPattern.ReplaceAllStringFunc(content, func(match string) string {
		text := strings.Trim(match, "*")
		return color.New(color.Italic).Sprint(text)
	})
}

// formatQuotes 格式化引用
func (f *Formatter) formatQuotes(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		if strings.HasPrefix(line, "> ") {
			quoted := strings.TrimPrefix(line, "> ")
			result = append(result, f.colorizer.Dim("│ ")+f.colorizer.Quote(quoted))
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// formatLists 格式化列表
func (f *Formatter) formatLists(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// 無序列表
		if matched, _ := regexp.MatchString(`^\s*[-*+]\s+`, line); matched {
			listPattern := regexp.MustCompile(`^(\s*)([-*+])(\s+)(.*)`)
			if parts := listPattern.FindStringSubmatch(line); len(parts) == 5 {
				indent := parts[1]
				bullet := f.colorizer.Info("•")
				space := parts[3]
				text := parts[4]
				result = append(result, indent+bullet+space+text)
			} else {
				result = append(result, line)
			}
		} else if matched, _ := regexp.MatchString(`^\s*\d+\.\s+`, line); matched {
			// 有序列表
			listPattern := regexp.MustCompile(`^(\s*)(\d+\.)(\s+)(.*)`)
			if parts := listPattern.FindStringSubmatch(line); len(parts) == 5 {
				indent := parts[1]
				number := f.colorizer.Info(parts[2])
				space := parts[3]
				text := parts[4]
				result = append(result, indent+number+space+text)
			} else {
				result = append(result, line)
			}
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// FormatTable 格式化表格
func (f *Formatter) FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	// 計算每列的最大寬度
	columnWidths := make([]int, len(headers))
	for i, header := range headers {
		columnWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(columnWidths) && len(cell) > columnWidths[i] {
				columnWidths[i] = len(cell)
			}
		}
	}

	var result strings.Builder

	// 頂部邊框
	result.WriteString("┌")
	for i, width := range columnWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(columnWidths)-1 {
			result.WriteString("┬")
		}
	}
	result.WriteString("┐\n")

	// 標題行
	result.WriteString("│")
	for i, header := range headers {
		paddedHeader := fmt.Sprintf(" %-*s ", columnWidths[i], header)
		result.WriteString(f.colorizer.TableHeader(paddedHeader))
		result.WriteString("│")
	}
	result.WriteString("\n")

	// 分隔線
	result.WriteString("├")
	for i, width := range columnWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(columnWidths)-1 {
			result.WriteString("┼")
		}
	}
	result.WriteString("┤\n")

	// 資料行
	for rowIndex, row := range rows {
		result.WriteString("│")
		for i, cell := range row {
			if i < len(columnWidths) {
				paddedCell := fmt.Sprintf(" %-*s ", columnWidths[i], cell)
				if rowIndex%2 == 0 {
					result.WriteString(f.colorizer.TableRow(paddedCell))
				} else {
					result.WriteString(f.colorizer.TableAltRow(paddedCell))
				}
				result.WriteString("│")
			}
		}
		result.WriteString("\n")
	}

	// 底部邊框
	result.WriteString("└")
	for i, width := range columnWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(columnWidths)-1 {
			result.WriteString("┴")
		}
	}
	result.WriteString("┘")

	return result.String()
}

// FormatProgressBar 格式化進度條
func (f *Formatter) FormatProgressBar(current, total int, label string) string {
	if total == 0 {
		return ""
	}

	percentage := float64(current) / float64(total) * 100
	filled := int(percentage / 5) // 每5%一個字符

	bar := strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)

	return fmt.Sprintf("%s [%s] %d/%d (%.1f%%)",
		label,
		bar, // Use the bar variable instead of colorizer.Progress
		current,
		total,
		percentage,
	)
}

// WrapText 文字換行
func (f *Formatter) WrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+1+len(word) <= width {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}
