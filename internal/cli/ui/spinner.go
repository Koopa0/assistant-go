package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Spinner 表示載入指示器
type Spinner struct {
	message  string
	frames   []string
	interval time.Duration
	active   bool
	mutex    sync.Mutex
	done     chan bool
}

// NewSpinner 建立新的載入指示器
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		frames: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
		interval: 100 * time.Millisecond,
		active:   false,
		done:     make(chan bool),
	}
}

// Start 開始顯示載入指示器
func (s *Spinner) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.active {
		return
	}

	s.active = true
	go s.spin()
}

// Stop 停止載入指示器
func (s *Spinner) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.active {
		return
	}

	s.active = false
	s.done <- true

	// 清除載入指示器
	fmt.Print("\r\033[K")
}

// UpdateMessage 更新顯示訊息
func (s *Spinner) UpdateMessage(message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.message = message
}

// spin 內部載入動畫迴圈
func (s *Spinner) spin() {
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	frameIndex := 0

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			if !s.active {
				return
			}

			frame := cyan(s.frames[frameIndex])
			message := dim(s.message)

			fmt.Printf("\r%s %s", frame, message)

			frameIndex = (frameIndex + 1) % len(s.frames)
		}
	}
}

// SetFrames 設定自訂動畫幀
func (s *Spinner) SetFrames(frames []string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.frames = frames
}

// SetInterval 設定動畫間隔
func (s *Spinner) SetInterval(interval time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.interval = interval
}

// 預定義的載入動畫樣式
var (
	// DotsSpinner 點點載入動畫
	DotsSpinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	// CircleSpinner 圓圈載入動畫
	CircleSpinner = []string{"◐", "◓", "◑", "◒"}

	// ArrowSpinner 箭頭載入動畫
	ArrowSpinner = []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}

	// BarSpinner 條狀載入動畫
	BarSpinner = []string{"▁", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃"}

	// StarSpinner 星星載入動畫
	StarSpinner = []string{"✶", "✸", "✹", "✺", "✹", "✷"}
)

// NewDotsSpinner 建立點點載入動畫
func NewDotsSpinner(message string) *Spinner {
	spinner := NewSpinner(message)
	spinner.SetFrames(DotsSpinner)
	return spinner
}

// NewCircleSpinner 建立圓圈載入動畫
func NewCircleSpinner(message string) *Spinner {
	spinner := NewSpinner(message)
	spinner.SetFrames(CircleSpinner)
	spinner.SetInterval(200 * time.Millisecond)
	return spinner
}

// NewArrowSpinner 建立箭頭載入動畫
func NewArrowSpinner(message string) *Spinner {
	spinner := NewSpinner(message)
	spinner.SetFrames(ArrowSpinner)
	spinner.SetInterval(150 * time.Millisecond)
	return spinner
}
