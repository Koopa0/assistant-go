package context

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WorkspaceContext tracks the current workspace state and project context
type WorkspaceContext struct {
	activeProjects map[string]*ProjectState
	openFiles      map[string]*FileContext
	recentChanges  *ChangeHistory
	dependencies   *DependencyGraph
	currentProject string
	workingDir     string
	logger         *slog.Logger
	mu             sync.RWMutex
}

// ProjectState represents the state of a project
type ProjectState struct {
	ID           string
	Name         string
	Path         string
	Type         ProjectType
	Language     string
	Framework    string
	LastAccessed time.Time
	FileTree     *FileTree
	GitInfo      *GitInfo
	Dependencies []Dependency
	Metadata     map[string]interface{}
}

// ProjectType defines the type of project
type ProjectType string

const (
	ProjectTypeGo         ProjectType = "go"
	ProjectTypeJavaScript ProjectType = "javascript"
	ProjectTypeTypeScript ProjectType = "typescript"
	ProjectTypePython     ProjectType = "python"
	ProjectTypeDocker     ProjectType = "docker"
	ProjectTypeKubernetes ProjectType = "kubernetes"
	ProjectTypeUnknown    ProjectType = "unknown"
)

// FileContext represents context about an open file
type FileContext struct {
	Path         string
	Language     string
	Size         int64
	LastModified time.Time
	LastAccessed time.Time
	ChangeCount  int
	IsModified   bool
	Content      *FileContent
}

// FileContent represents the semantic content of a file
type FileContent struct {
	Type       string
	Imports    []string
	Functions  []string
	Classes    []string
	Interfaces []string
	Variables  []string
	Comments   []string
	TODOs      []string
	Complexity int
}

// ChangeHistory tracks recent changes in the workspace
type ChangeHistory struct {
	Changes  []Change
	MaxItems int
	mu       sync.RWMutex
}

// Change represents a workspace change
type Change struct {
	ID        string
	Type      ChangeType
	Path      string
	Timestamp time.Time
	Details   map[string]interface{}
}

// ChangeType defines types of workspace changes
type ChangeType string

const (
	ChangeFileCreated   ChangeType = "file_created"
	ChangeFileModified  ChangeType = "file_modified"
	ChangeFileDeleted   ChangeType = "file_deleted"
	ChangeFileOpened    ChangeType = "file_opened"
	ChangeFileClosed    ChangeType = "file_closed"
	ChangeProjectSwitch ChangeType = "project_switch"
)

// DependencyGraph represents project dependencies
type DependencyGraph struct {
	Dependencies map[string]Dependency
	mu           sync.RWMutex
}

// Dependency represents a project dependency
type Dependency struct {
	Name    string
	Version string
	Type    string
	Path    string
}

// WorkspaceInfo contains relevant workspace context for a request
type WorkspaceInfo struct {
	ActiveProject      string
	OpenFiles          []string
	RecentChanges      []Change
	CurrentDir         string
	GitBranch          string
	UncommittedChanges bool
	ProjectType        ProjectType
	Language           string
	Framework          string
	Dependencies       []Dependency
}

// WorkspaceState represents current workspace state
type WorkspaceState struct {
	Projects       map[string]*ProjectState
	OpenFiles      map[string]*FileContext
	CurrentProject string
	WorkingDir     string
	LastUpdate     time.Time
}

// GitInfo represents git repository information
type GitInfo struct {
	Branch             string
	LastCommit         string
	UncommittedChanges bool
	RemoteURL          string
}

// FileTree represents the structure of a project
type FileTree struct {
	Root  *FileNode
	Files map[string]*FileNode
}

// FileNode represents a file or directory in the tree
type FileNode struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	Modified time.Time
	Children []*FileNode
	Parent   *FileNode
}

// NewWorkspaceContext creates a new workspace context
func NewWorkspaceContext(logger *slog.Logger) (*WorkspaceContext, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	return &WorkspaceContext{
		activeProjects: make(map[string]*ProjectState),
		openFiles:      make(map[string]*FileContext),
		recentChanges: &ChangeHistory{
			Changes:  make([]Change, 0),
			MaxItems: 100,
		},
		dependencies: &DependencyGraph{Dependencies: make(map[string]Dependency)},
		workingDir:   workingDir,
		logger:       logger,
	}, nil
}

// GetRelevantContext extracts relevant workspace context for a request
func (wc *WorkspaceContext) GetRelevantContext(ctx context.Context, request Request) (WorkspaceInfo, error) {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	// Detect current project if not set
	if wc.currentProject == "" {
		if err := wc.detectCurrentProject(); err != nil {
			wc.logger.Warn("Failed to detect current project", slog.Any("error", err))
		}
	}

	var info WorkspaceInfo

	// Get active project info
	if wc.currentProject != "" {
		if project, exists := wc.activeProjects[wc.currentProject]; exists {
			info.ActiveProject = project.Name
			info.ProjectType = project.Type
			info.Language = project.Language
			info.Framework = project.Framework
			info.Dependencies = project.Dependencies

			// Get git info
			if project.GitInfo != nil {
				info.GitBranch = project.GitInfo.Branch
				info.UncommittedChanges = project.GitInfo.UncommittedChanges
			}
		}
	}

	// Get open files
	info.OpenFiles = make([]string, 0, len(wc.openFiles))
	for path := range wc.openFiles {
		info.OpenFiles = append(info.OpenFiles, path)
	}

	// Get recent changes (last 10)
	wc.recentChanges.mu.RLock()
	changeCount := len(wc.recentChanges.Changes)
	if changeCount > 10 {
		info.RecentChanges = wc.recentChanges.Changes[changeCount-10:]
	} else {
		info.RecentChanges = wc.recentChanges.Changes
	}
	wc.recentChanges.mu.RUnlock()

	info.CurrentDir = wc.workingDir

	return info, nil
}

// ProcessUpdate processes a workspace context update
func (wc *WorkspaceContext) ProcessUpdate(ctx context.Context, update ContextUpdate) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	switch update.Type {
	case WorkspaceChange:
		return wc.handleWorkspaceChange(update)
	case FileActivity:
		return wc.handleFileActivity(update)
	case ProjectSwitch:
		return wc.handleProjectSwitch(update)
	default:
		return fmt.Errorf("unsupported update type: %s", update.Type)
	}
}

// GetCurrentState returns the current workspace state
func (wc *WorkspaceContext) GetCurrentState() WorkspaceState {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	return WorkspaceState{
		Projects:       wc.activeProjects,
		OpenFiles:      wc.openFiles,
		CurrentProject: wc.currentProject,
		WorkingDir:     wc.workingDir,
		LastUpdate:     time.Now(),
	}
}

// OpenFile tracks an opened file
func (wc *WorkspaceContext) OpenFile(filePath string) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	fileContext := &FileContext{
		Path:         absPath,
		Language:     wc.detectLanguage(absPath),
		Size:         fileInfo.Size(),
		LastModified: fileInfo.ModTime(),
		LastAccessed: time.Now(),
		ChangeCount:  0,
		IsModified:   false,
	}

	// Analyze file content
	content, err := wc.analyzeFileContent(absPath)
	if err != nil {
		wc.logger.Warn("Failed to analyze file content", slog.String("path", absPath), slog.Any("error", err))
	} else {
		fileContext.Content = content
	}

	wc.openFiles[absPath] = fileContext

	// Record the change
	wc.addChange(Change{
		ID:        fmt.Sprintf("file_open_%d", time.Now().UnixNano()),
		Type:      ChangeFileOpened,
		Path:      absPath,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"language": fileContext.Language,
			"size":     fileContext.Size,
		},
	})

	wc.logger.Debug("File opened and tracked",
		slog.String("path", absPath),
		slog.String("language", fileContext.Language))

	return nil
}

// CloseFile stops tracking a file
func (wc *WorkspaceContext) CloseFile(filePath string) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if _, exists := wc.openFiles[absPath]; exists {
		delete(wc.openFiles, absPath)

		wc.addChange(Change{
			ID:        fmt.Sprintf("file_close_%d", time.Now().UnixNano()),
			Type:      ChangeFileClosed,
			Path:      absPath,
			Timestamp: time.Now(),
		})

		wc.logger.Debug("File closed and untracked", slog.String("path", absPath))
	}

	return nil
}

// detectCurrentProject attempts to detect the current project
func (wc *WorkspaceContext) detectCurrentProject() error {
	// Look for common project indicators
	projectIndicators := []string{
		"go.mod",           // Go project
		"package.json",     // Node.js project
		"requirements.txt", // Python project
		"Dockerfile",       // Docker project
		".git",             // Git repository
	}

	for _, indicator := range projectIndicators {
		if _, err := os.Stat(filepath.Join(wc.workingDir, indicator)); err == nil {
			// Found project indicator, create or update project
			project, err := wc.analyzeProject(wc.workingDir, indicator)
			if err != nil {
				return fmt.Errorf("failed to analyze project: %w", err)
			}

			wc.activeProjects[project.ID] = project
			wc.currentProject = project.ID

			wc.logger.Info("Detected current project",
				slog.String("project_id", project.ID),
				slog.String("name", project.Name),
				slog.String("type", string(project.Type)))

			return nil
		}
	}

	return fmt.Errorf("no project detected in current directory")
}

// analyzeProject analyzes a project directory
func (wc *WorkspaceContext) analyzeProject(projectPath, indicator string) (*ProjectState, error) {
	projectName := filepath.Base(projectPath)
	projectID := fmt.Sprintf("%s_%s", projectName, strings.ReplaceAll(projectPath, "/", "_"))

	project := &ProjectState{
		ID:           projectID,
		Name:         projectName,
		Path:         projectPath,
		LastAccessed: time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// Detect project type and language
	switch indicator {
	case "go.mod":
		project.Type = ProjectTypeGo
		project.Language = "go"
		project.Framework = wc.detectGoFramework(projectPath)
	case "package.json":
		project.Type = ProjectTypeJavaScript
		project.Language = "javascript"
		project.Framework = wc.detectJSFramework(projectPath)
	case "requirements.txt":
		project.Type = ProjectTypePython
		project.Language = "python"
		project.Framework = wc.detectPythonFramework(projectPath)
	case "Dockerfile":
		project.Type = ProjectTypeDocker
	case ".git":
		project.GitInfo = wc.analyzeGitInfo(projectPath)
	}

	return project, nil
}

// detectLanguage detects the programming language of a file
func (wc *WorkspaceContext) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":    "go",
		".js":    "javascript",
		".ts":    "typescript",
		".py":    "python",
		".java":  "java",
		".cpp":   "cpp",
		".c":     "c",
		".rb":    "ruby",
		".php":   "php",
		".rs":    "rust",
		".kt":    "kotlin",
		".swift": "swift",
		".yaml":  "yaml",
		".yml":   "yaml",
		".json":  "json",
		".xml":   "xml",
		".sql":   "sql",
		".md":    "markdown",
		".txt":   "text",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "unknown"
}

// analyzeFileContent analyzes the content of a file for semantic information
func (wc *WorkspaceContext) analyzeFileContent(filePath string) (*FileContent, error) {
	// For now, return basic content structure
	// This would be enhanced with actual AST parsing in a full implementation
	return &FileContent{
		Type:       wc.detectLanguage(filePath),
		Imports:    []string{},
		Functions:  []string{},
		Classes:    []string{},
		Interfaces: []string{},
		Variables:  []string{},
		Comments:   []string{},
		TODOs:      []string{},
		Complexity: 1,
	}, nil
}

// Framework detection methods (simplified implementations)
func (wc *WorkspaceContext) detectGoFramework(projectPath string) string {
	// Check for common Go frameworks
	if _, err := os.Stat(filepath.Join(projectPath, "gin")); err == nil {
		return "gin"
	}
	if _, err := os.Stat(filepath.Join(projectPath, "echo")); err == nil {
		return "echo"
	}
	return "standard"
}

func (wc *WorkspaceContext) detectJSFramework(projectPath string) string {
	// Check package.json for framework dependencies
	return "unknown"
}

func (wc *WorkspaceContext) detectPythonFramework(projectPath string) string {
	// Check requirements.txt for framework dependencies
	return "unknown"
}

// analyzeGitInfo analyzes git repository information
func (wc *WorkspaceContext) analyzeGitInfo(projectPath string) *GitInfo {
	// Simplified git info - would use git commands in full implementation
	return &GitInfo{
		Branch:             "main",
		LastCommit:         "unknown",
		UncommittedChanges: false,
		RemoteURL:          "unknown",
	}
}

// Helper methods
func (wc *WorkspaceContext) handleWorkspaceChange(update ContextUpdate) error {
	// Handle workspace-level changes
	return nil
}

func (wc *WorkspaceContext) handleFileActivity(update ContextUpdate) error {
	// Handle file activity changes
	return nil
}

func (wc *WorkspaceContext) handleProjectSwitch(update ContextUpdate) error {
	// Handle project switching
	return nil
}

func (wc *WorkspaceContext) addChange(change Change) {
	wc.recentChanges.mu.Lock()
	defer wc.recentChanges.mu.Unlock()

	wc.recentChanges.Changes = append(wc.recentChanges.Changes, change)

	// Keep only the most recent changes
	if len(wc.recentChanges.Changes) > wc.recentChanges.MaxItems {
		wc.recentChanges.Changes = wc.recentChanges.Changes[1:]
	}
}

// Close shuts down the workspace context
func (wc *WorkspaceContext) Close(ctx context.Context) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	wc.logger.Info("Shutting down workspace context")

	// Clear all tracked data
	wc.activeProjects = nil
	wc.openFiles = nil
	wc.recentChanges = nil
	wc.dependencies = nil

	return nil
}
