package ui

import (
	"os"
	"strings"

	"github.com/fatih/color"
)

// Colorizer 提供顏色主題和樣式
type Colorizer struct {
	theme          string
	noColor        bool
	titleColor     *color.Color
	subtitleColor  *color.Color
	successColor   *color.Color
	errorColor     *color.Color
	warningColor   *color.Color
	infoColor      *color.Color
	dimColor       *color.Color
	boldColor      *color.Color
	assistantColor *color.Color
	userColor      *color.Color
	systemColor    *color.Color
	valueColor     *color.Color
	helpColor      *color.Color
}

// NewColorizer 建立新的顏色管理器
func NewColorizer(theme string) *Colorizer {
	// 檢查是否支援顏色
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"

	colorizer := &Colorizer{
		theme:   theme,
		noColor: noColor,
	}

	colorizer.setupTheme()
	return colorizer
}

// setupTheme 設定主題顏色
func (c *Colorizer) setupTheme() {
	if c.noColor {
		// 如果不支援顏色，所有顏色都設為無樣式
		c.titleColor = color.New()
		c.subtitleColor = color.New()
		c.successColor = color.New()
		c.errorColor = color.New()
		c.warningColor = color.New()
		c.infoColor = color.New()
		c.dimColor = color.New()
		c.boldColor = color.New()
		c.assistantColor = color.New()
		c.userColor = color.New()
		c.systemColor = color.New()
		c.valueColor = color.New()
		c.helpColor = color.New()
		return
	}

	switch c.theme {
	case "light":
		c.setupLightTheme()
	case "dark":
		c.setupDarkTheme()
	case "auto":
		if c.isDarkTerminal() {
			c.setupDarkTheme()
		} else {
			c.setupLightTheme()
		}
	default:
		c.setupDarkTheme() // 預設為深色主題
	}
}

// setupDarkTheme 設定深色主題
func (c *Colorizer) setupDarkTheme() {
	c.titleColor = color.New(color.FgCyan, color.Bold)
	c.subtitleColor = color.New(color.FgBlue)
	c.successColor = color.New(color.FgGreen)
	c.errorColor = color.New(color.FgRed, color.Bold)
	c.warningColor = color.New(color.FgYellow)
	c.infoColor = color.New(color.FgCyan)
	c.dimColor = color.New(color.Faint)
	c.boldColor = color.New(color.Bold)
	c.assistantColor = color.New(color.FgBlue, color.Bold)
	c.userColor = color.New(color.FgGreen)
	c.systemColor = color.New(color.FgMagenta)
	c.valueColor = color.New(color.FgYellow)
	c.helpColor = color.New(color.FgWhite)
}

// setupLightTheme 設定淺色主題
func (c *Colorizer) setupLightTheme() {
	c.titleColor = color.New(color.FgBlue, color.Bold)
	c.subtitleColor = color.New(color.FgBlue)
	c.successColor = color.New(color.FgGreen, color.Bold)
	c.errorColor = color.New(color.FgRed, color.Bold)
	c.warningColor = color.New(color.FgRed)
	c.infoColor = color.New(color.FgBlue)
	c.dimColor = color.New(color.Faint)
	c.boldColor = color.New(color.Bold)
	c.assistantColor = color.New(color.FgBlue, color.Bold)
	c.userColor = color.New(color.FgGreen, color.Bold)
	c.systemColor = color.New(color.FgMagenta)
	c.valueColor = color.New(color.FgRed)
	c.helpColor = color.New(color.FgBlack)
}

// isDarkTerminal 檢測終端是否為深色背景
func (c *Colorizer) isDarkTerminal() bool {
	// 簡單的檢測邏輯，可以根據需要改進
	term := strings.ToLower(os.Getenv("TERM"))
	colorTerm := strings.ToLower(os.Getenv("COLORTERM"))

	// 大多數現代終端預設為深色背景
	return !strings.Contains(term, "light") && !strings.Contains(colorTerm, "light")
}

// 顏色方法

// Title 標題顏色
func (c *Colorizer) Title(text string) string {
	return c.titleColor.Sprint(text)
}

// Subtitle 副標題顏色
func (c *Colorizer) Subtitle(text string) string {
	return c.subtitleColor.Sprint(text)
}

// Success 成功訊息顏色
func (c *Colorizer) Success(text string) string {
	return c.successColor.Sprint(text)
}

// Error 錯誤訊息顏色
func (c *Colorizer) Error(text string) string {
	return c.errorColor.Sprint(text)
}

// Warning 警告訊息顏色
func (c *Colorizer) Warning(text string) string {
	return c.warningColor.Sprint(text)
}

// Info 資訊訊息顏色
func (c *Colorizer) Info(text string) string {
	return c.infoColor.Sprint(text)
}

// Dim 淡化文字顏色
func (c *Colorizer) Dim(text string) string {
	return c.dimColor.Sprint(text)
}

// Bold 粗體文字
func (c *Colorizer) Bold(text string) string {
	return c.boldColor.Sprint(text)
}

// Assistant 助手訊息顏色
func (c *Colorizer) Assistant(text string) string {
	return c.assistantColor.Sprint(text)
}

// User 使用者訊息顏色
func (c *Colorizer) User(text string) string {
	return c.userColor.Sprint(text)
}

// System 系統訊息顏色
func (c *Colorizer) System(text string) string {
	return c.systemColor.Sprint(text)
}

// Value 數值顏色
func (c *Colorizer) Value(text string) string {
	return c.valueColor.Sprint(text)
}

// Help 幫助文字顏色
func (c *Colorizer) Help(text string) string {
	return c.helpColor.Sprint(text)
}

// Divider 分隔線
func (c *Colorizer) Divider() string {
	line := strings.Repeat("─", 60)
	return c.dimColor.Sprint(line)
}

// Highlight 高亮文字
func (c *Colorizer) Highlight(text string) string {
	highlightColor := color.New(color.BgYellow, color.FgBlack)
	return highlightColor.Sprint(text)
}

// Code 程式碼樣式
func (c *Colorizer) Code(text string) string {
	codeColor := color.New(color.FgCyan, color.Faint)
	return codeColor.Sprint(text)
}

// Link 連結樣式
func (c *Colorizer) Link(text string) string {
	linkColor := color.New(color.FgBlue, color.Underline)
	return linkColor.Sprint(text)
}

// Quote 引用樣式
func (c *Colorizer) Quote(text string) string {
	quoteColor := color.New(color.FgYellow, color.Italic)
	return quoteColor.Sprint(text)
}

// Progress 進度條顏色
func (c *Colorizer) Progress(completed, total int) string {
	if total == 0 {
		return ""
	}

	percentage := float64(completed) / float64(total) * 100
	filled := int(percentage / 5) // 每5%一個字符，總共20個字符

	progressBar := strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)

	if percentage >= 100 {
		return c.successColor.Sprintf("[%s] %.0f%%", progressBar, percentage)
	} else if percentage >= 50 {
		return c.infoColor.Sprintf("[%s] %.0f%%", progressBar, percentage)
	} else {
		return c.warningColor.Sprintf("[%s] %.0f%%", progressBar, percentage)
	}
}

// Status 狀態顏色
func (c *Colorizer) Status(status string) string {
	switch strings.ToLower(status) {
	case "online", "active", "running", "success", "ok", "healthy":
		return c.Success(status)
	case "offline", "inactive", "stopped", "error", "failed", "unhealthy":
		return c.Error(status)
	case "pending", "loading", "warning", "degraded":
		return c.Warning(status)
	default:
		return c.Info(status)
	}
}

// Table 表格樣式
func (c *Colorizer) TableHeader(text string) string {
	return c.boldColor.Sprint(text)
}

func (c *Colorizer) TableRow(text string) string {
	return text
}

func (c *Colorizer) TableAltRow(text string) string {
	return c.dimColor.Sprint(text)
}

// Badge 徽章樣式
func (c *Colorizer) Badge(text, badgeType string) string {
	var badgeColor *color.Color

	switch strings.ToLower(badgeType) {
	case "success", "green":
		badgeColor = color.New(color.BgGreen, color.FgWhite, color.Bold)
	case "error", "red":
		badgeColor = color.New(color.BgRed, color.FgWhite, color.Bold)
	case "warning", "yellow":
		badgeColor = color.New(color.BgYellow, color.FgBlack, color.Bold)
	case "info", "blue":
		badgeColor = color.New(color.BgBlue, color.FgWhite, color.Bold)
	default:
		badgeColor = color.New(color.BgWhite, color.FgBlack, color.Bold)
	}

	return badgeColor.Sprintf(" %s ", text)
}

// GetTheme 獲取當前主題
func (c *Colorizer) GetTheme() string {
	return c.theme
}

// SetTheme 設定主題
func (c *Colorizer) SetTheme(theme string) {
	c.theme = theme
	c.setupTheme()
}

// IsColorEnabled 檢查是否啟用顏色
func (c *Colorizer) IsColorEnabled() bool {
	return !c.noColor
}

// DisableColor 停用顏色
func (c *Colorizer) DisableColor() {
	c.noColor = true
	c.setupTheme()
}

// EnableColor 啟用顏色
func (c *Colorizer) EnableColor() {
	c.noColor = false
	c.setupTheme()
}
