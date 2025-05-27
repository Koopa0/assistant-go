package agents

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant-go/internal/config"
)

// InfrastructureAgent specializes in infrastructure management and troubleshooting
type InfrastructureAgent struct {
	*BaseAgent
	k8sManager     *KubernetesManager
	dockerManager  *DockerManager
	logAnalyzer    *LogAnalyzer
	troubleshooter *Troubleshooter
}

// NewInfrastructureAgent creates a new infrastructure agent
func NewInfrastructureAgent(llm llms.Model, config config.LangChain, logger *slog.Logger) *InfrastructureAgent {
	base := NewBaseAgent(AgentTypeInfrastructure, llm, config, logger)

	agent := &InfrastructureAgent{
		BaseAgent:      base,
		k8sManager:     NewKubernetesManager(logger),
		dockerManager:  NewDockerManager(logger),
		logAnalyzer:    NewLogAnalyzer(llm, logger),
		troubleshooter: NewTroubleshooter(llm, logger),
	}

	// Add infrastructure-specific capabilities
	agent.initializeCapabilities()

	return agent
}

// initializeCapabilities sets up the infrastructure agent's capabilities
func (i *InfrastructureAgent) initializeCapabilities() {
	capabilities := []AgentCapability{
		{
			Name:        "kubernetes_management",
			Description: "Manage Kubernetes clusters, pods, services, and deployments",
			Parameters: map[string]interface{}{
				"action":    "string (list|describe|create|delete|scale|logs)",
				"resource":  "string (pods|services|deployments|nodes)",
				"namespace": "string",
				"name":      "string",
			},
		},
		{
			Name:        "docker_management",
			Description: "Manage Docker containers, images, and networks",
			Parameters: map[string]interface{}{
				"action":   "string (list|run|stop|remove|logs|inspect)",
				"resource": "string (containers|images|networks|volumes)",
				"name":     "string",
				"image":    "string",
				"ports":    "[]string",
			},
		},
		{
			Name:        "log_analysis",
			Description: "Analyze application and system logs for issues and patterns",
			Parameters: map[string]interface{}{
				"log_source":   "string (kubernetes|docker|system|application)",
				"time_range":   "string",
				"log_level":    "string (error|warn|info|debug)",
				"search_terms": "[]string",
			},
		},
		{
			Name:        "troubleshooting",
			Description: "Diagnose and troubleshoot infrastructure issues",
			Parameters: map[string]interface{}{
				"issue_type":  "string (performance|connectivity|resource|deployment)",
				"symptoms":    "[]string",
				"environment": "string (development|staging|production)",
				"urgency":     "string (low|medium|high|critical)",
			},
		},
		{
			Name:        "resource_monitoring",
			Description: "Monitor and analyze resource usage and performance",
			Parameters: map[string]interface{}{
				"resource_type": "string (cpu|memory|disk|network)",
				"time_range":    "string",
				"threshold":     "number",
				"alert_level":   "string",
			},
		},
		{
			Name:        "deployment_assistance",
			Description: "Help with application deployments and rollbacks",
			Parameters: map[string]interface{}{
				"deployment_type": "string (kubernetes|docker|helm)",
				"action":          "string (deploy|rollback|scale|update)",
				"environment":     "string",
				"version":         "string",
			},
		},
	}

	for _, capability := range capabilities {
		i.AddCapability(capability)
	}
}

// executeSteps implements specialized infrastructure agent logic
func (i *InfrastructureAgent) executeSteps(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Analyze the request to determine the infrastructure task
	taskType, err := i.analyzeRequestType(request.Query)
	if err != nil {
		return "", nil, fmt.Errorf("failed to analyze request type: %w", err)
	}

	i.logger.Debug("Infrastructure task identified",
		slog.String("task_type", taskType),
		slog.String("query", request.Query))

	// Execute based on task type
	switch taskType {
	case "kubernetes_management":
		return i.executeKubernetesManagement(ctx, request, maxSteps)
	case "docker_management":
		return i.executeDockerManagement(ctx, request, maxSteps)
	case "log_analysis":
		return i.executeLogAnalysis(ctx, request, maxSteps)
	case "troubleshooting":
		return i.executeTroubleshooting(ctx, request, maxSteps)
	case "resource_monitoring":
		return i.executeResourceMonitoring(ctx, request, maxSteps)
	case "deployment_assistance":
		return i.executeDeploymentAssistance(ctx, request, maxSteps)
	default:
		return i.executeGeneralInfrastructure(ctx, request, maxSteps)
	}
}

// analyzeRequestType determines what type of infrastructure task is being requested
func (i *InfrastructureAgent) analyzeRequestType(query string) (string, error) {
	query = strings.ToLower(query)

	// Kubernetes patterns
	if strings.Contains(query, "kubernetes") || strings.Contains(query, "k8s") ||
		strings.Contains(query, "pod") || strings.Contains(query, "deployment") ||
		strings.Contains(query, "service") || strings.Contains(query, "kubectl") {
		return "kubernetes_management", nil
	}

	// Docker patterns
	if strings.Contains(query, "docker") || strings.Contains(query, "container") ||
		strings.Contains(query, "image") {
		return "docker_management", nil
	}

	// Log analysis patterns
	if strings.Contains(query, "log") || strings.Contains(query, "logs") ||
		strings.Contains(query, "analyze") && (strings.Contains(query, "error") || strings.Contains(query, "issue")) {
		return "log_analysis", nil
	}

	// Troubleshooting patterns
	if strings.Contains(query, "troubleshoot") || strings.Contains(query, "debug") ||
		strings.Contains(query, "issue") || strings.Contains(query, "problem") ||
		strings.Contains(query, "not working") || strings.Contains(query, "failing") {
		return "troubleshooting", nil
	}

	// Resource monitoring patterns
	if strings.Contains(query, "monitor") || strings.Contains(query, "resource") ||
		strings.Contains(query, "cpu") || strings.Contains(query, "memory") ||
		strings.Contains(query, "performance") {
		return "resource_monitoring", nil
	}

	// Deployment patterns
	if strings.Contains(query, "deploy") || strings.Contains(query, "rollback") ||
		strings.Contains(query, "scale") || strings.Contains(query, "update") {
		return "deployment_assistance", nil
	}

	return "general", nil
}

// executeKubernetesManagement handles Kubernetes operations
func (i *InfrastructureAgent) executeKubernetesManagement(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	steps := make([]AgentStep, 0)

	// Step 1: Parse Kubernetes command
	stepStart := time.Now()
	k8sCommand := i.parseKubernetesCommand(request.Query)

	step1 := AgentStep{
		StepNumber: 1,
		Action:     "parse_k8s_command",
		Input:      request.Query,
		Output:     fmt.Sprintf("Parsed command: %s", k8sCommand.Action),
		Reasoning:  "Parse Kubernetes operation from user request",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"action": k8sCommand.Action, "resource": k8sCommand.Resource},
	}
	steps = append(steps, step1)

	// Step 2: Execute Kubernetes operation
	stepStart = time.Now()
	result, err := i.k8sManager.ExecuteCommand(ctx, k8sCommand)
	if err != nil {
		return "", nil, fmt.Errorf("Kubernetes operation failed: %w", err)
	}

	step2 := AgentStep{
		StepNumber: 2,
		Action:     "execute_k8s_operation",
		Tool:       "kubernetes_manager",
		Input:      fmt.Sprintf("%s %s", k8sCommand.Action, k8sCommand.Resource),
		Output:     result.Summary,
		Reasoning:  "Execute Kubernetes operation",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"success": result.Success},
	}
	steps = append(steps, step2)

	// Step 3: Generate explanation using LLM
	stepStart = time.Now()
	prompt := i.buildK8sExplanationPrompt(k8sCommand, result)
	explanation, err := i.llm.Call(ctx, prompt, llms.WithMaxTokens(1500))
	if err != nil {
		return "", nil, fmt.Errorf("explanation generation failed: %w", err)
	}

	step3 := AgentStep{
		StepNumber: 3,
		Action:     "generate_explanation",
		Tool:       "llm",
		Input:      prompt,
		Output:     explanation,
		Reasoning:  "Generate user-friendly explanation of the operation",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"explanation_length": len(explanation)},
	}
	steps = append(steps, step3)

	// Build final response
	response := fmt.Sprintf("Kubernetes Operation Results:\n\n%s\n\nExplanation:\n%s", result.Details, explanation)

	return response, steps, nil
}

// executeDockerManagement handles Docker operations
func (i *InfrastructureAgent) executeDockerManagement(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	steps := make([]AgentStep, 0)

	// Step 1: Parse Docker command
	stepStart := time.Now()
	dockerCommand := i.parseDockerCommand(request.Query)

	step1 := AgentStep{
		StepNumber: 1,
		Action:     "parse_docker_command",
		Input:      request.Query,
		Output:     fmt.Sprintf("Parsed command: %s", dockerCommand.Action),
		Reasoning:  "Parse Docker operation from user request",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"action": dockerCommand.Action, "resource": dockerCommand.Resource},
	}
	steps = append(steps, step1)

	// Step 2: Execute Docker operation
	stepStart = time.Now()
	result, err := i.dockerManager.ExecuteCommand(ctx, dockerCommand)
	if err != nil {
		return "", nil, fmt.Errorf("Docker operation failed: %w", err)
	}

	step2 := AgentStep{
		StepNumber: 2,
		Action:     "execute_docker_operation",
		Tool:       "docker_manager",
		Input:      fmt.Sprintf("%s %s", dockerCommand.Action, dockerCommand.Resource),
		Output:     result.Summary,
		Reasoning:  "Execute Docker operation",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"success": result.Success},
	}
	steps = append(steps, step2)

	// Step 3: Generate explanation
	stepStart = time.Now()
	prompt := i.buildDockerExplanationPrompt(dockerCommand, result)
	explanation, err := i.llm.Call(ctx, prompt, llms.WithMaxTokens(1500))
	if err != nil {
		return "", nil, fmt.Errorf("explanation generation failed: %w", err)
	}

	step3 := AgentStep{
		StepNumber: 3,
		Action:     "generate_explanation",
		Tool:       "llm",
		Input:      prompt,
		Output:     explanation,
		Reasoning:  "Generate user-friendly explanation of the operation",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"explanation_length": len(explanation)},
	}
	steps = append(steps, step3)

	response := fmt.Sprintf("Docker Operation Results:\n\n%s\n\nExplanation:\n%s", result.Details, explanation)

	return response, steps, nil
}

// executeLogAnalysis analyzes logs for issues and patterns
func (i *InfrastructureAgent) executeLogAnalysis(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	steps := make([]AgentStep, 0)

	// Step 1: Identify log source and parameters
	stepStart := time.Now()
	logParams := i.parseLogAnalysisParams(request.Query)

	step1 := AgentStep{
		StepNumber: 1,
		Action:     "parse_log_params",
		Input:      request.Query,
		Output:     fmt.Sprintf("Log source: %s, Level: %s", logParams.Source, logParams.Level),
		Reasoning:  "Identify log analysis parameters",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"source": logParams.Source, "level": logParams.Level},
	}
	steps = append(steps, step1)

	// Step 2: Collect logs
	stepStart = time.Now()
	logs, err := i.logAnalyzer.CollectLogs(ctx, logParams)
	if err != nil {
		return "", nil, fmt.Errorf("log collection failed: %w", err)
	}

	step2 := AgentStep{
		StepNumber: 2,
		Action:     "collect_logs",
		Tool:       "log_analyzer",
		Input:      fmt.Sprintf("Source: %s", logParams.Source),
		Output:     fmt.Sprintf("Collected %d log entries", len(logs.Entries)),
		Reasoning:  "Collect logs from specified source",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"log_count": len(logs.Entries)},
	}
	steps = append(steps, step2)

	// Step 3: Analyze logs using LLM
	stepStart = time.Now()
	prompt := i.buildLogAnalysisPrompt(logs, logParams)
	analysis, err := i.llm.Call(ctx, prompt, llms.WithMaxTokens(2000))
	if err != nil {
		return "", nil, fmt.Errorf("log analysis failed: %w", err)
	}

	step3 := AgentStep{
		StepNumber: 3,
		Action:     "analyze_logs",
		Tool:       "llm",
		Input:      prompt,
		Output:     analysis,
		Reasoning:  "Analyze logs for patterns and issues",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"analysis_length": len(analysis)},
	}
	steps = append(steps, step3)

	response := fmt.Sprintf("Log Analysis Results:\n\n%s", analysis)

	return response, steps, nil
}

// executeTroubleshooting performs infrastructure troubleshooting
func (i *InfrastructureAgent) executeTroubleshooting(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Simplified implementation - delegate to general infrastructure handling
	return i.executeGeneralInfrastructure(ctx, request, maxSteps)
}

// executeResourceMonitoring monitors resource usage
func (i *InfrastructureAgent) executeResourceMonitoring(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Simplified implementation - delegate to general infrastructure handling
	return i.executeGeneralInfrastructure(ctx, request, maxSteps)
}

// executeDeploymentAssistance helps with deployments
func (i *InfrastructureAgent) executeDeploymentAssistance(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Simplified implementation - delegate to general infrastructure handling
	return i.executeGeneralInfrastructure(ctx, request, maxSteps)
}

// executeGeneralInfrastructure handles general infrastructure queries
func (i *InfrastructureAgent) executeGeneralInfrastructure(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Fallback to base agent implementation with infrastructure context
	prompt := fmt.Sprintf("As an infrastructure expert, please help with: %s", request.Query)
	result, err := i.llm.Call(ctx, prompt, llms.WithMaxTokens(2000))
	if err != nil {
		return "", nil, fmt.Errorf("general infrastructure query failed: %w", err)
	}

	step := AgentStep{
		StepNumber: 1,
		Action:     "general_infrastructure_assistance",
		Tool:       "llm",
		Input:      prompt,
		Output:     result,
		Reasoning:  "Provide general infrastructure assistance",
		Duration:   time.Millisecond * 100, // Placeholder
		Metadata:   map[string]interface{}{"response_length": len(result)},
	}

	return result, []AgentStep{step}, nil
}

// Helper methods

func (i *InfrastructureAgent) parseKubernetesCommand(query string) *K8sCommand {
	command := &K8sCommand{
		Action:    "list",
		Resource:  "pods",
		Namespace: "default",
	}

	query = strings.ToLower(query)

	// Parse action
	if strings.Contains(query, "create") {
		command.Action = "create"
	} else if strings.Contains(query, "delete") {
		command.Action = "delete"
	} else if strings.Contains(query, "describe") {
		command.Action = "describe"
	} else if strings.Contains(query, "logs") {
		command.Action = "logs"
	} else if strings.Contains(query, "scale") {
		command.Action = "scale"
	}

	// Parse resource
	if strings.Contains(query, "deployment") {
		command.Resource = "deployments"
	} else if strings.Contains(query, "service") {
		command.Resource = "services"
	} else if strings.Contains(query, "node") {
		command.Resource = "nodes"
	}

	return command
}

func (i *InfrastructureAgent) parseDockerCommand(query string) *DockerCommand {
	command := &DockerCommand{
		Action:   "list",
		Resource: "containers",
	}

	query = strings.ToLower(query)

	// Parse action
	if strings.Contains(query, "run") {
		command.Action = "run"
	} else if strings.Contains(query, "stop") {
		command.Action = "stop"
	} else if strings.Contains(query, "remove") || strings.Contains(query, "rm") {
		command.Action = "remove"
	} else if strings.Contains(query, "logs") {
		command.Action = "logs"
	} else if strings.Contains(query, "inspect") {
		command.Action = "inspect"
	}

	// Parse resource
	if strings.Contains(query, "image") {
		command.Resource = "images"
	} else if strings.Contains(query, "network") {
		command.Resource = "networks"
	} else if strings.Contains(query, "volume") {
		command.Resource = "volumes"
	}

	return command
}

func (i *InfrastructureAgent) parseLogAnalysisParams(query string) *LogAnalysisParams {
	params := &LogAnalysisParams{
		Source:    "application",
		Level:     "error",
		TimeRange: "1h",
	}

	query = strings.ToLower(query)

	// Parse source
	if strings.Contains(query, "kubernetes") || strings.Contains(query, "k8s") {
		params.Source = "kubernetes"
	} else if strings.Contains(query, "docker") {
		params.Source = "docker"
	} else if strings.Contains(query, "system") {
		params.Source = "system"
	}

	// Parse level
	if strings.Contains(query, "debug") {
		params.Level = "debug"
	} else if strings.Contains(query, "info") {
		params.Level = "info"
	} else if strings.Contains(query, "warn") {
		params.Level = "warn"
	}

	return params
}

func (i *InfrastructureAgent) buildK8sExplanationPrompt(command *K8sCommand, result *OperationResult) string {
	return fmt.Sprintf(`Explain the following Kubernetes operation:

Command: %s %s
Namespace: %s
Success: %v
Result: %s

Please provide:
1. What this operation does
2. Why it might be useful
3. Any important considerations
4. Next steps if applicable

Explanation:`, command.Action, command.Resource, command.Namespace, result.Success, result.Summary)
}

func (i *InfrastructureAgent) buildDockerExplanationPrompt(command *DockerCommand, result *OperationResult) string {
	return fmt.Sprintf(`Explain the following Docker operation:

Command: %s %s
Success: %v
Result: %s

Please provide:
1. What this operation does
2. Why it might be useful
3. Any important considerations
4. Next steps if applicable

Explanation:`, command.Action, command.Resource, result.Success, result.Summary)
}

func (i *InfrastructureAgent) buildLogAnalysisPrompt(logs *LogCollection, params *LogAnalysisParams) string {
	logSample := ""
	if len(logs.Entries) > 0 {
		// Include first few log entries as sample
		for idx, entry := range logs.Entries {
			if idx >= 5 { // Limit to first 5 entries
				break
			}
			logSample += fmt.Sprintf("[%s] %s: %s\n", entry.Timestamp, entry.Level, entry.Message)
		}
	}

	return fmt.Sprintf(`Analyze the following logs:

Source: %s
Level: %s
Time Range: %s
Total Entries: %d

Sample Log Entries:
%s

Please provide:
1. Summary of log patterns
2. Any errors or issues identified
3. Recommendations for investigation
4. Potential root causes

Analysis:`, params.Source, params.Level, params.TimeRange, len(logs.Entries), logSample)
}

// Supporting types

type K8sCommand struct {
	Action    string            `json:"action"`
	Resource  string            `json:"resource"`
	Namespace string            `json:"namespace"`
	Name      string            `json:"name,omitempty"`
	Options   map[string]string `json:"options,omitempty"`
}

type DockerCommand struct {
	Action   string            `json:"action"`
	Resource string            `json:"resource"`
	Name     string            `json:"name,omitempty"`
	Image    string            `json:"image,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
}

type LogAnalysisParams struct {
	Source      string   `json:"source"`
	Level       string   `json:"level"`
	TimeRange   string   `json:"time_range"`
	SearchTerms []string `json:"search_terms,omitempty"`
}

type OperationResult struct {
	Success bool   `json:"success"`
	Summary string `json:"summary"`
	Details string `json:"details"`
	Error   string `json:"error,omitempty"`
}

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type LogCollection struct {
	Entries []LogEntry `json:"entries"`
	Source  string     `json:"source"`
	Count   int        `json:"count"`
}

// KubernetesManager manages Kubernetes operations
type KubernetesManager struct {
	logger *slog.Logger
}

func NewKubernetesManager(logger *slog.Logger) *KubernetesManager {
	return &KubernetesManager{logger: logger}
}

func (k *KubernetesManager) ExecuteCommand(ctx context.Context, command *K8sCommand) (*OperationResult, error) {
	// Simplified implementation - would integrate with actual Kubernetes client
	result := &OperationResult{
		Success: true,
		Summary: fmt.Sprintf("Executed %s %s", command.Action, command.Resource),
		Details: fmt.Sprintf("Successfully performed %s operation on %s in namespace %s",
			command.Action, command.Resource, command.Namespace),
	}

	k.logger.Debug("Kubernetes command executed",
		slog.String("action", command.Action),
		slog.String("resource", command.Resource),
		slog.String("namespace", command.Namespace))

	return result, nil
}

// DockerManager manages Docker operations
type DockerManager struct {
	logger *slog.Logger
}

func NewDockerManager(logger *slog.Logger) *DockerManager {
	return &DockerManager{logger: logger}
}

func (d *DockerManager) ExecuteCommand(ctx context.Context, command *DockerCommand) (*OperationResult, error) {
	// Simplified implementation - would integrate with actual Docker client
	result := &OperationResult{
		Success: true,
		Summary: fmt.Sprintf("Executed %s %s", command.Action, command.Resource),
		Details: fmt.Sprintf("Successfully performed %s operation on %s",
			command.Action, command.Resource),
	}

	d.logger.Debug("Docker command executed",
		slog.String("action", command.Action),
		slog.String("resource", command.Resource))

	return result, nil
}

// LogAnalyzer analyzes logs from various sources
type LogAnalyzer struct {
	llm    llms.Model
	logger *slog.Logger
}

func NewLogAnalyzer(llm llms.Model, logger *slog.Logger) *LogAnalyzer {
	return &LogAnalyzer{llm: llm, logger: logger}
}

func (l *LogAnalyzer) CollectLogs(ctx context.Context, params *LogAnalysisParams) (*LogCollection, error) {
	// Simplified implementation - would integrate with actual log sources
	collection := &LogCollection{
		Entries: make([]LogEntry, 0),
		Source:  params.Source,
		Count:   0,
	}

	// Add some sample log entries for demonstration
	sampleEntries := []LogEntry{
		{
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     "error",
			Message:   "Connection timeout to database",
			Source:    params.Source,
		},
		{
			Timestamp: time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			Level:     "warn",
			Message:   "High memory usage detected",
			Source:    params.Source,
		},
	}

	collection.Entries = sampleEntries
	collection.Count = len(sampleEntries)

	l.logger.Debug("Logs collected",
		slog.String("source", params.Source),
		slog.Int("count", collection.Count))

	return collection, nil
}

// Troubleshooter helps diagnose infrastructure issues
type Troubleshooter struct {
	llm    llms.Model
	logger *slog.Logger
}

func NewTroubleshooter(llm llms.Model, logger *slog.Logger) *Troubleshooter {
	return &Troubleshooter{llm: llm, logger: logger}
}
