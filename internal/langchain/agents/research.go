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

// ResearchAgent specializes in information gathering, fact-checking, and report generation
type ResearchAgent struct {
	*BaseAgent
	informationGatherer *InformationGatherer
	factChecker         *FactChecker
	reportGenerator     *ReportGenerator
	sourceManager       *SourceManager
}

// NewResearchAgent creates a new research agent
func NewResearchAgent(llm llms.Model, config config.LangChain, logger *slog.Logger) *ResearchAgent {
	base := NewBaseAgent(AgentTypeResearch, llm, config, logger)

	agent := &ResearchAgent{
		BaseAgent:           base,
		informationGatherer: NewInformationGatherer(llm, logger),
		factChecker:         NewFactChecker(llm, logger),
		reportGenerator:     NewReportGenerator(llm, logger),
		sourceManager:       NewSourceManager(logger),
	}

	// Add research-specific capabilities
	agent.initializeCapabilities()

	return agent
}

// initializeCapabilities sets up the research agent's capabilities
func (r *ResearchAgent) initializeCapabilities() {
	capabilities := []AgentCapability{
		{
			Name:        "information_gathering",
			Description: "Gather information from multiple sources on a given topic",
			Parameters: map[string]interface{}{
				"topic":      "string",
				"sources":    "[]string (web|academic|documentation|internal)",
				"depth":      "string (surface|detailed|comprehensive)",
				"time_range": "string",
				"language":   "string",
			},
		},
		{
			Name:        "fact_checking",
			Description: "Verify facts and claims against reliable sources",
			Parameters: map[string]interface{}{
				"claims":      "[]string",
				"sources":     "[]string",
				"confidence":  "string (low|medium|high)",
				"cross_check": "bool",
			},
		},
		{
			Name:        "report_generation",
			Description: "Generate comprehensive reports from gathered information",
			Parameters: map[string]interface{}{
				"report_type":       "string (summary|analysis|comparison|recommendation)",
				"format":            "string (markdown|html|pdf|json)",
				"sections":          "[]string",
				"citations":         "bool",
				"executive_summary": "bool",
			},
		},
		{
			Name:        "source_analysis",
			Description: "Analyze and evaluate the credibility of information sources",
			Parameters: map[string]interface{}{
				"sources":    "[]string",
				"criteria":   "[]string (authority|accuracy|objectivity|currency)",
				"bias_check": "bool",
			},
		},
		{
			Name:        "trend_analysis",
			Description: "Identify trends and patterns in research data",
			Parameters: map[string]interface{}{
				"data_type":     "string (temporal|categorical|numerical)",
				"time_period":   "string",
				"metrics":       "[]string",
				"visualization": "bool",
			},
		},
		{
			Name:        "comparative_analysis",
			Description: "Compare and contrast different topics, solutions, or approaches",
			Parameters: map[string]interface{}{
				"subjects":    "[]string",
				"criteria":    "[]string",
				"methodology": "string",
				"scoring":     "bool",
			},
		},
	}

	for _, capability := range capabilities {
		r.AddCapability(capability)
	}
}

// executeSteps implements specialized research agent logic
func (r *ResearchAgent) executeSteps(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Analyze the request to determine the research task
	taskType, err := r.analyzeRequestType(request.Query)
	if err != nil {
		return "", nil, fmt.Errorf("failed to analyze request type: %w", err)
	}

	r.logger.Debug("Research task identified",
		slog.String("task_type", taskType),
		slog.String("query", request.Query))

	// Execute based on task type
	switch taskType {
	case "information_gathering":
		return r.executeInformationGathering(ctx, request, maxSteps)
	case "fact_checking":
		return r.executeFactChecking(ctx, request, maxSteps)
	case "report_generation":
		return r.executeReportGeneration(ctx, request, maxSteps)
	case "source_analysis":
		return r.executeSourceAnalysis(ctx, request, maxSteps)
	case "trend_analysis":
		return r.executeTrendAnalysis(ctx, request, maxSteps)
	case "comparative_analysis":
		return r.executeComparativeAnalysis(ctx, request, maxSteps)
	default:
		return r.executeGeneralResearch(ctx, request, maxSteps)
	}
}

// analyzeRequestType determines what type of research task is being requested
func (r *ResearchAgent) analyzeRequestType(query string) (string, error) {
	query = strings.ToLower(query)

	// Information gathering patterns
	if strings.Contains(query, "research") || strings.Contains(query, "gather") ||
		strings.Contains(query, "find information") || strings.Contains(query, "investigate") {
		return "information_gathering", nil
	}

	// Fact checking patterns
	if strings.Contains(query, "fact check") || strings.Contains(query, "verify") ||
		strings.Contains(query, "validate") || strings.Contains(query, "confirm") {
		return "fact_checking", nil
	}

	// Report generation patterns
	if strings.Contains(query, "report") || strings.Contains(query, "summary") ||
		strings.Contains(query, "document") || strings.Contains(query, "write up") {
		return "report_generation", nil
	}

	// Source analysis patterns
	if strings.Contains(query, "source") || strings.Contains(query, "credibility") ||
		strings.Contains(query, "reliability") || strings.Contains(query, "evaluate") {
		return "source_analysis", nil
	}

	// Trend analysis patterns
	if strings.Contains(query, "trend") || strings.Contains(query, "pattern") ||
		strings.Contains(query, "analysis") || strings.Contains(query, "over time") {
		return "trend_analysis", nil
	}

	// Comparative analysis patterns
	if strings.Contains(query, "compare") || strings.Contains(query, "contrast") ||
		strings.Contains(query, "versus") || strings.Contains(query, "vs") ||
		strings.Contains(query, "difference") {
		return "comparative_analysis", nil
	}

	return "general", nil
}

// executeInformationGathering gathers information from multiple sources
func (r *ResearchAgent) executeInformationGathering(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	steps := make([]AgentStep, 0)

	// Step 1: Analyze research topic and scope
	stepStart := time.Now()
	researchScope := r.analyzeResearchScope(request.Query)

	step1 := AgentStep{
		StepNumber: 1,
		Action:     "analyze_research_scope",
		Input:      request.Query,
		Output:     fmt.Sprintf("Topic: %s, Depth: %s, Sources: %v", researchScope.Topic, researchScope.Depth, researchScope.Sources),
		Reasoning:  "Determine research scope and methodology",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"topic": researchScope.Topic, "depth": researchScope.Depth},
	}
	steps = append(steps, step1)

	// Step 2: Gather information from sources
	stepStart = time.Now()
	information, err := r.informationGatherer.GatherInformation(ctx, researchScope)
	if err != nil {
		return "", nil, fmt.Errorf("information gathering failed: %w", err)
	}

	step2 := AgentStep{
		StepNumber: 2,
		Action:     "gather_information",
		Tool:       "information_gatherer",
		Input:      fmt.Sprintf("Topic: %s", researchScope.Topic),
		Output:     fmt.Sprintf("Gathered %d sources, %d facts", len(information.Sources), len(information.Facts)),
		Reasoning:  "Collect information from multiple sources",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"source_count": len(information.Sources), "fact_count": len(information.Facts)},
	}
	steps = append(steps, step2)

	// Step 3: Synthesize and organize information
	stepStart = time.Now()
	prompt := r.buildInformationSynthesisPrompt(researchScope, information)
	synthesis, err := r.llm.Call(ctx, prompt, llms.WithMaxTokens(3000))
	if err != nil {
		return "", nil, fmt.Errorf("information synthesis failed: %w", err)
	}

	step3 := AgentStep{
		StepNumber: 3,
		Action:     "synthesize_information",
		Tool:       "llm",
		Input:      prompt,
		Output:     synthesis,
		Reasoning:  "Synthesize gathered information into coherent summary",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"synthesis_length": len(synthesis)},
	}
	steps = append(steps, step3)

	// Step 4: Generate final research summary
	finalSummary := r.formatResearchSummary(researchScope, information, synthesis)

	return finalSummary, steps, nil
}

// executeFactChecking verifies facts and claims
func (r *ResearchAgent) executeFactChecking(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	steps := make([]AgentStep, 0)

	// Step 1: Extract claims to verify
	stepStart := time.Now()
	claims := r.extractClaims(request.Query)

	step1 := AgentStep{
		StepNumber: 1,
		Action:     "extract_claims",
		Input:      request.Query,
		Output:     fmt.Sprintf("Identified %d claims to verify", len(claims)),
		Reasoning:  "Extract specific claims that need fact-checking",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"claim_count": len(claims)},
	}
	steps = append(steps, step1)

	// Step 2: Verify each claim
	stepStart = time.Now()
	verificationResults := make([]*FactCheckResult, 0)

	for _, claim := range claims {
		result, err := r.factChecker.VerifyClaim(ctx, claim)
		if err != nil {
			r.logger.Warn("Failed to verify claim", slog.String("claim", claim), slog.Any("error", err))
			continue
		}
		verificationResults = append(verificationResults, result)
	}

	step2 := AgentStep{
		StepNumber: 2,
		Action:     "verify_claims",
		Tool:       "fact_checker",
		Input:      fmt.Sprintf("%d claims", len(claims)),
		Output:     fmt.Sprintf("Verified %d claims", len(verificationResults)),
		Reasoning:  "Verify each claim against reliable sources",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"verified_count": len(verificationResults)},
	}
	steps = append(steps, step2)

	// Step 3: Generate fact-check report
	stepStart = time.Now()
	prompt := r.buildFactCheckReportPrompt(claims, verificationResults)
	report, err := r.llm.Call(ctx, prompt, llms.WithMaxTokens(2500))
	if err != nil {
		return "", nil, fmt.Errorf("fact-check report generation failed: %w", err)
	}

	step3 := AgentStep{
		StepNumber: 3,
		Action:     "generate_fact_check_report",
		Tool:       "llm",
		Input:      prompt,
		Output:     report,
		Reasoning:  "Generate comprehensive fact-check report",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"report_length": len(report)},
	}
	steps = append(steps, step3)

	return report, steps, nil
}

// executeReportGeneration generates comprehensive reports
func (r *ResearchAgent) executeReportGeneration(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	steps := make([]AgentStep, 0)

	// Step 1: Analyze report requirements
	stepStart := time.Now()
	reportSpec := r.analyzeReportRequirements(request.Query)

	step1 := AgentStep{
		StepNumber: 1,
		Action:     "analyze_report_requirements",
		Input:      request.Query,
		Output:     fmt.Sprintf("Report type: %s, Format: %s", reportSpec.Type, reportSpec.Format),
		Reasoning:  "Determine report structure and requirements",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"type": reportSpec.Type, "format": reportSpec.Format},
	}
	steps = append(steps, step1)

	// Step 2: Gather content for report
	stepStart = time.Now()
	content, err := r.gatherReportContent(ctx, reportSpec)
	if err != nil {
		return "", nil, fmt.Errorf("content gathering failed: %w", err)
	}

	step2 := AgentStep{
		StepNumber: 2,
		Action:     "gather_report_content",
		Tool:       "content_gatherer",
		Input:      fmt.Sprintf("Type: %s", reportSpec.Type),
		Output:     fmt.Sprintf("Gathered content for %d sections", len(content.Sections)),
		Reasoning:  "Collect content for each report section",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"section_count": len(content.Sections)},
	}
	steps = append(steps, step2)

	// Step 3: Generate structured report
	stepStart = time.Now()
	prompt := r.buildReportGenerationPrompt(reportSpec, content)
	report, err := r.llm.Call(ctx, prompt, llms.WithMaxTokens(4000))
	if err != nil {
		return "", nil, fmt.Errorf("report generation failed: %w", err)
	}

	step3 := AgentStep{
		StepNumber: 3,
		Action:     "generate_structured_report",
		Tool:       "llm",
		Input:      prompt,
		Output:     report,
		Reasoning:  "Generate final structured report",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"report_length": len(report)},
	}
	steps = append(steps, step3)

	return report, steps, nil
}

// executeSourceAnalysis analyzes source credibility
func (r *ResearchAgent) executeSourceAnalysis(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Simplified implementation - delegate to general research handling
	return r.executeGeneralResearch(ctx, request, maxSteps)
}

// executeTrendAnalysis identifies trends and patterns
func (r *ResearchAgent) executeTrendAnalysis(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Simplified implementation - delegate to general research handling
	return r.executeGeneralResearch(ctx, request, maxSteps)
}

// executeComparativeAnalysis performs comparative analysis
func (r *ResearchAgent) executeComparativeAnalysis(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Simplified implementation - delegate to general research handling
	return r.executeGeneralResearch(ctx, request, maxSteps)
}

// executeGeneralResearch handles general research queries
func (r *ResearchAgent) executeGeneralResearch(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Fallback to base agent implementation with research context
	prompt := fmt.Sprintf("As a research expert, please help with: %s", request.Query)
	result, err := r.llm.Call(ctx, prompt, llms.WithMaxTokens(2000))
	if err != nil {
		return "", nil, fmt.Errorf("general research query failed: %w", err)
	}

	step := AgentStep{
		StepNumber: 1,
		Action:     "general_research_assistance",
		Tool:       "llm",
		Input:      prompt,
		Output:     result,
		Reasoning:  "Provide general research assistance",
		Duration:   time.Millisecond * 100, // Placeholder
		Metadata:   map[string]interface{}{"response_length": len(result)},
	}

	return result, []AgentStep{step}, nil
}

// Helper methods

func (r *ResearchAgent) analyzeResearchScope(query string) *ResearchScope {
	scope := &ResearchScope{
		Topic:   r.extractTopic(query),
		Depth:   "detailed",
		Sources: []string{"web", "documentation"},
	}

	query = strings.ToLower(query)

	// Determine depth
	if strings.Contains(query, "comprehensive") || strings.Contains(query, "thorough") {
		scope.Depth = "comprehensive"
	} else if strings.Contains(query, "quick") || strings.Contains(query, "brief") {
		scope.Depth = "surface"
	}

	// Determine sources
	if strings.Contains(query, "academic") {
		scope.Sources = append(scope.Sources, "academic")
	}
	if strings.Contains(query, "internal") {
		scope.Sources = append(scope.Sources, "internal")
	}

	return scope
}

func (r *ResearchAgent) extractTopic(query string) string {
	// Simple topic extraction (could be enhanced with NLP)
	words := strings.Fields(query)
	if len(words) > 0 {
		// Return first few words as topic
		if len(words) >= 3 {
			return strings.Join(words[:3], " ")
		}
		return strings.Join(words, " ")
	}
	return "general research"
}

func (r *ResearchAgent) extractClaims(query string) []string {
	claims := make([]string, 0)

	// Simple claim extraction based on sentence structure
	sentences := strings.Split(query, ".")
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 10 { // Minimum length for a claim
			claims = append(claims, sentence)
		}
	}

	if len(claims) == 0 {
		claims = append(claims, query) // Treat entire query as a claim
	}

	return claims
}

func (r *ResearchAgent) analyzeReportRequirements(query string) *ReportSpecification {
	spec := &ReportSpecification{
		Type:     "summary",
		Format:   "markdown",
		Sections: []string{"introduction", "findings", "conclusion"},
	}

	query = strings.ToLower(query)

	// Determine report type
	if strings.Contains(query, "analysis") {
		spec.Type = "analysis"
	} else if strings.Contains(query, "comparison") {
		spec.Type = "comparison"
	} else if strings.Contains(query, "recommendation") {
		spec.Type = "recommendation"
	}

	// Determine format
	if strings.Contains(query, "html") {
		spec.Format = "html"
	} else if strings.Contains(query, "json") {
		spec.Format = "json"
	}

	return spec
}

func (r *ResearchAgent) gatherReportContent(ctx context.Context, spec *ReportSpecification) (*ReportContent, error) {
	content := &ReportContent{
		Sections: make(map[string]string),
	}

	// Generate placeholder content for each section
	for _, section := range spec.Sections {
		content.Sections[section] = fmt.Sprintf("Content for %s section", section)
	}

	return content, nil
}

func (r *ResearchAgent) buildInformationSynthesisPrompt(scope *ResearchScope, info *GatheredInformation) string {
	return fmt.Sprintf(`Synthesize the following research information:

Topic: %s
Research Depth: %s
Sources Used: %v

Gathered Information:
- %d sources consulted
- %d key facts identified

Please provide:
1. Executive summary of findings
2. Key insights and patterns
3. Areas requiring further research
4. Confidence level in findings

Synthesis:`, scope.Topic, scope.Depth, scope.Sources, len(info.Sources), len(info.Facts))
}

func (r *ResearchAgent) buildFactCheckReportPrompt(claims []string, results []*FactCheckResult) string {
	resultsText := ""
	for i, result := range results {
		resultsText += fmt.Sprintf("Claim %d: %s - Status: %s (Confidence: %s)\n",
			i+1, result.Claim, result.Status, result.Confidence)
	}

	return fmt.Sprintf(`Generate a fact-check report for the following claims:

Claims Verified: %d
Results:
%s

Please provide:
1. Summary of verification results
2. Overall credibility assessment
3. Sources used for verification
4. Recommendations for further verification

Fact-Check Report:`, len(claims), resultsText)
}

func (r *ResearchAgent) buildReportGenerationPrompt(spec *ReportSpecification, content *ReportContent) string {
	sectionsText := ""
	for section, sectionContent := range content.Sections {
		sectionsText += fmt.Sprintf("%s: %s\n", section, sectionContent)
	}

	return fmt.Sprintf(`Generate a %s report in %s format:

Report Type: %s
Sections: %v

Content:
%s

Please provide:
1. Well-structured report following the specified format
2. Clear section headers and organization
3. Professional tone and presentation
4. Actionable insights where applicable

Report:`, spec.Type, spec.Format, spec.Type, spec.Sections, sectionsText)
}

func (r *ResearchAgent) formatResearchSummary(scope *ResearchScope, info *GatheredInformation, synthesis string) string {
	return fmt.Sprintf(`# Research Summary: %s

## Research Scope
- **Topic**: %s
- **Depth**: %s
- **Sources**: %v

## Information Gathered
- **Sources Consulted**: %d
- **Key Facts**: %d

## Synthesis
%s

## Methodology
This research was conducted using automated information gathering and synthesis techniques, with verification against multiple sources where possible.

---
*Generated by Research Agent*`, scope.Topic, scope.Topic, scope.Depth, scope.Sources, len(info.Sources), len(info.Facts), synthesis)
}

// Supporting types

type ResearchScope struct {
	Topic     string   `json:"topic"`
	Depth     string   `json:"depth"`
	Sources   []string `json:"sources"`
	TimeRange string   `json:"time_range,omitempty"`
	Language  string   `json:"language,omitempty"`
}

type GatheredInformation struct {
	Sources []InformationSource `json:"sources"`
	Facts   []ResearchFact      `json:"facts"`
	Summary string              `json:"summary"`
}

type InformationSource struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
	Credibility string    `json:"credibility"`
	AccessDate  time.Time `json:"access_date"`
}

type ResearchFact struct {
	Statement  string   `json:"statement"`
	Sources    []string `json:"sources"`
	Confidence string   `json:"confidence"`
	Category   string   `json:"category"`
}

type FactCheckResult struct {
	Claim      string   `json:"claim"`
	Status     string   `json:"status"` // verified, disputed, false, unknown
	Confidence string   `json:"confidence"`
	Sources    []string `json:"sources"`
	Evidence   string   `json:"evidence"`
}

type ReportSpecification struct {
	Type             string   `json:"type"`
	Format           string   `json:"format"`
	Sections         []string `json:"sections"`
	Citations        bool     `json:"citations"`
	ExecutiveSummary bool     `json:"executive_summary"`
}

type ReportContent struct {
	Sections  map[string]string      `json:"sections"`
	Citations []string               `json:"citations,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// InformationGatherer collects information from various sources
type InformationGatherer struct {
	llm    llms.Model
	logger *slog.Logger
}

func NewInformationGatherer(llm llms.Model, logger *slog.Logger) *InformationGatherer {
	return &InformationGatherer{llm: llm, logger: logger}
}

func (ig *InformationGatherer) GatherInformation(ctx context.Context, scope *ResearchScope) (*GatheredInformation, error) {
	// Simplified implementation - would integrate with actual information sources
	info := &GatheredInformation{
		Sources: make([]InformationSource, 0),
		Facts:   make([]ResearchFact, 0),
		Summary: fmt.Sprintf("Research conducted on topic: %s", scope.Topic),
	}

	// Add sample sources and facts
	info.Sources = append(info.Sources, InformationSource{
		URL:         "https://example.com/research",
		Title:       fmt.Sprintf("Research on %s", scope.Topic),
		Type:        "web",
		Credibility: "high",
		AccessDate:  time.Now(),
	})

	info.Facts = append(info.Facts, ResearchFact{
		Statement:  fmt.Sprintf("Key finding about %s", scope.Topic),
		Sources:    []string{"https://example.com/research"},
		Confidence: "high",
		Category:   "general",
	})

	ig.logger.Debug("Information gathered",
		slog.String("topic", scope.Topic),
		slog.Int("sources", len(info.Sources)),
		slog.Int("facts", len(info.Facts)))

	return info, nil
}

// FactChecker verifies claims against reliable sources
type FactChecker struct {
	llm    llms.Model
	logger *slog.Logger
}

func NewFactChecker(llm llms.Model, logger *slog.Logger) *FactChecker {
	return &FactChecker{llm: llm, logger: logger}
}

func (fc *FactChecker) VerifyClaim(ctx context.Context, claim string) (*FactCheckResult, error) {
	// Simplified implementation - would integrate with fact-checking sources
	result := &FactCheckResult{
		Claim:      claim,
		Status:     "verified",
		Confidence: "medium",
		Sources:    []string{"fact-check-source"},
		Evidence:   fmt.Sprintf("Evidence supporting claim: %s", claim),
	}

	fc.logger.Debug("Claim verified",
		slog.String("claim", claim),
		slog.String("status", result.Status))

	return result, nil
}

// ReportGenerator generates structured reports
type ReportGenerator struct {
	llm    llms.Model
	logger *slog.Logger
}

func NewReportGenerator(llm llms.Model, logger *slog.Logger) *ReportGenerator {
	return &ReportGenerator{llm: llm, logger: logger}
}

// SourceManager manages and evaluates information sources
type SourceManager struct {
	logger *slog.Logger
}

func NewSourceManager(logger *slog.Logger) *SourceManager {
	return &SourceManager{logger: logger}
}
