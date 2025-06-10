# Prompt Engineering Guide for Go Development

**Last Updated**: 2025-01-07  
**Version**: 1.0

## 🎯 Overview

This guide explains how Assistant incorporates Go best practices and the CLAUDE-ARCHITECTURE principles into its prompt system, making it an exceptional tool for Go developers.

## 📚 Core Principles Integration

### 1. Go Philosophy in Prompts

Our prompts embed Go's core values:

```go
const codeReviewPrompt = `
As a Go expert following the language's philosophy:
- Clear is better than clever
- Explicit is better than implicit  
- Errors are values, not exceptions
- Simplicity is the ultimate sophistication

Review this code focusing on:
1. Idiomatic Go patterns
2. Error handling completeness
3. Concurrency safety
4. Interface design (small and focused)

Code to review:
%s
`
```

### 2. CLAUDE-ARCHITECTURE Principles

Each prompt incorporates specific guidelines:

```go
const refactoringPrompt = `
Following CLAUDE-ARCHITECTURE.md principles:

1. Package Organization:
   - Feature-based, not layer-based
   - Package names are singular nouns
   - Like Go stdlib: "user" not "users"

2. Interface Design:
   - Accept interfaces, return structs
   - Define interfaces where consumed
   - Keep interfaces small (1-3 methods ideal)

3. Error Handling:
   - Always wrap with context: fmt.Errorf("operation: %w", err)
   - Never ignore errors
   - Create custom error types for business logic

Refactor this code:
%s
`
```

## 🧠 Intelligent Prompt Templates

### Code Analysis Template

```go
type CodeAnalysisPrompt struct {
    BasePrompt string
    Context    AnalysisContext
}

func (p *CodeAnalysisPrompt) Generate() string {
    return fmt.Sprintf(`
You are a senior Go engineer analyzing code with expertise in:
- Go standard library patterns
- PostgreSQL optimization with pgx v5
- Type-safe SQL with sqlc
- Production-grade concurrent systems

Context:
- Project Type: %s
- Current Package: %s
- Dependencies: %v
- Team Conventions: %v

Analyze this code for:
1. Go idioms and best practices
2. Performance implications
3. Concurrency safety
4. Error handling completeness
5. Test coverage opportunities

Always reference Go standard library patterns when suggesting improvements.
For example:
- io.Reader/Writer for interface design
- http.Handler for middleware patterns
- context for cancellation and values

Code:
%s

Provide specific, actionable feedback with code examples.
`,
        p.Context.ProjectType,
        p.Context.CurrentPackage,
        p.Context.Dependencies,
        p.Context.Conventions,
        p.Context.Code,
    )
}
```

### Pattern Detection Template

```go
const patternDetectionPrompt = `
As a Go expert studying the standard library for patterns:

Identify patterns in this code similar to:
1. io package: Small, composable interfaces
2. http package: Middleware and handler patterns  
3. sync package: Concurrency primitives
4. encoding packages: Marshal/Unmarshal patterns

Look for:
- Repeated code structures
- Common error handling patterns
- Interface usage patterns
- Concurrency patterns

Code to analyze:
%s

Output format:
1. Pattern name and description
2. Similar stdlib pattern
3. Suggested improvements
4. Example implementation
`
```

## 🔧 Context-Aware Prompts

### Workspace Context Integration

```go
func buildContextualPrompt(base string, ctx *WorkspaceContext) string {
    enhancedPrompt := fmt.Sprintf(`
Current workspace context:
- Language: Go %s
- Project Type: %s
- Database: %s with %s
- Testing: %s
- Main frameworks: %v

User preferences:
- Error style: %s
- Test style: %s
- Naming convention: %s

%s`,
        ctx.GoVersion,
        ctx.ProjectType,
        ctx.Database.Type,
        ctx.Database.ORM,
        ctx.TestFramework,
        ctx.Frameworks,
        ctx.Preferences.ErrorStyle,
        ctx.Preferences.TestStyle,
        ctx.Preferences.NamingConvention,
        base,
    )
    
    return enhancedPrompt
}
```

### Historical Context

```go
type HistoricalPromptEnhancer struct {
    memory *MemoryStore
}

func (h *HistoricalPromptEnhancer) Enhance(prompt string) string {
    recentPatterns := h.memory.GetRecentPatterns(5)
    commonMistakes := h.memory.GetCommonMistakes(3)
    
    return fmt.Sprintf(`
Based on recent interactions:
- Common patterns used: %v
- Previous mistakes to avoid: %v
- Preferred solutions: %v

%s`,
        recentPatterns,
        commonMistakes,
        h.memory.GetPreferredSolutions(),
        prompt,
    )
}
```

## 📋 Specialized Prompts

### 1. Error Handling Prompt

```go
const errorHandlingPrompt = `
Following Go's "errors are values" philosophy and CLAUDE-ARCHITECTURE guidelines:

Rules:
1. Always wrap errors with context: fmt.Errorf("failed to X: %w", err)
2. Create custom error types for business logic
3. Use errors.Is() and errors.As() for checking
4. Never panic in libraries
5. Log errors once at the top level

Analyze error handling in:
%s

Suggest improvements following patterns from:
- database/sql for error variables
- os package for path errors
- net package for network errors
`
```

### 2. Testing Prompt

```go
const testGenerationPrompt = `
Generate tests following Go testing best practices:

1. Table-driven tests (default approach)
2. Test behavior, not implementation
3. Use real dependencies when possible
4. Clear test names: Test<Type>_<Method>_<Scenario>

Reference patterns from Go stdlib tests:
- strings package: Excellent table-driven examples
- net/http: httptest usage
- io: Interface testing patterns

User preference: %s testing framework

Generate tests for:
%s

Include:
- Happy path
- Error cases  
- Edge cases
- Concurrent usage (if applicable)
`
```

### 3. Performance Optimization Prompt

```go
const performancePrompt = `
Optimize following Go performance best practices:

First, measure with benchmarks:
func BenchmarkXxx(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // code to benchmark
    }
}

Common optimizations:
1. Preallocate slices: make([]T, 0, expectedSize)
2. Use sync.Pool for temporary objects
3. Avoid string concatenation in loops (use strings.Builder)
4. Profile before optimizing (pprof)

Never optimize without measurement.

Code to analyze:
%s

Provide:
1. Benchmark code
2. Identified bottlenecks
3. Suggested optimizations
4. Expected improvements
`
```

## 🎨 Advanced Prompt Patterns

### 1. Progressive Enhancement

```go
type ProgressivePrompt struct {
    levels []string
}

func NewProgressivePrompt() *ProgressivePrompt {
    return &ProgressivePrompt{
        levels: []string{
            "Basic: Make it work correctly",
            "Intermediate: Make it idiomatic Go",
            "Advanced: Optimize for production",
            "Expert: Scale to millions of users",
        },
    }
}

func (p *ProgressivePrompt) Generate(level int, code string) string {
    if level >= len(p.levels) {
        level = len(p.levels) - 1
    }
    
    return fmt.Sprintf(`
Enhancement Level: %s

Previous levels completed: %v

Enhance this code:
%s
`,
        p.levels[level],
        p.levels[:level],
        code,
    )
}
```

### 2. Comparison Prompts

```go
const comparisonPrompt = `
Compare these implementations like a Go expert would:

Consider:
1. Idiomatic Go style
2. Performance characteristics
3. Maintainability
4. Testability
5. Concurrency safety

Reference similar comparisons from Go stdlib:
- sync.Map vs map with mutex
- bytes.Buffer vs strings.Builder
- channel vs mutex for coordination

Implementation A:
%s

Implementation B:
%s

Provide:
- Pros/cons of each
- Performance implications  
- Recommended choice with reasoning
- Hybrid approach if applicable
`
```

## 🔄 Dynamic Prompt Generation

### Context-Sensitive Prompts

```go
func generateDynamicPrompt(ctx context.Context, task Task) string {
    user := getUserFromContext(ctx)
    project := getProjectFromContext(ctx)
    
    promptBuilder := &strings.Builder{}
    
    // Base expertise
    promptBuilder.WriteString("As a Go expert ")
    
    // Add specific expertise based on project
    switch project.Type {
    case "microservice":
        promptBuilder.WriteString("specializing in microservices, ")
    case "cli":
        promptBuilder.WriteString("specializing in CLI tools, ")
    case "library":
        promptBuilder.WriteString("specializing in library design, ")
    }
    
    // Add user-specific context
    if user.Experience == "senior" {
        promptBuilder.WriteString("providing advanced insights:\n\n")
    } else {
        promptBuilder.WriteString("providing clear explanations:\n\n")
    }
    
    // Add task-specific instructions
    promptBuilder.WriteString(task.GenerateInstructions())
    
    return promptBuilder.String()
}
```

### Learning-Based Adaptation

```go
type AdaptivePromptGenerator struct {
    learningService *learning.Service
}

func (g *AdaptivePromptGenerator) Generate(base string) string {
    // Get user's common mistakes
    mistakes := g.learningService.GetCommonMistakes(limit: 5)
    
    // Get successful patterns
    patterns := g.learningService.GetSuccessfulPatterns(limit: 5)
    
    // Adapt prompt based on learning
    adapted := fmt.Sprintf(`
Based on your coding patterns:
- Common issues to watch for: %v
- Successful patterns to apply: %v

%s

Pay special attention to:
%s
`,
        mistakes,
        patterns,
        base,
        g.generateFocusAreas(mistakes),
    )
    
    return adapted
}
```

## 📊 Prompt Effectiveness Metrics

### Tracking Success

```go
type PromptMetrics struct {
    PromptID       string
    UserSatisfaction float64
    CodeQuality    float64
    TimeToComplete time.Duration
    Revisions      int
}

func (m *PromptMetrics) Score() float64 {
    return (m.UserSatisfaction * 0.4) + 
           (m.CodeQuality * 0.4) + 
           ((1.0 / float64(m.Revisions+1)) * 0.2)
}
```

### A/B Testing Prompts

```go
func selectPromptVariant(userID string, promptType string) string {
    // Use consistent hashing for user
    variant := hashUserToVariant(userID, promptType)
    
    switch variant {
    case "A":
        return promptVariantA
    case "B":
        return promptVariantB
    default:
        return promptControl
    }
}
```

## 🎓 Best Practices for Prompt Engineering

### 1. Be Specific About Go Version

```go
prompt := fmt.Sprintf(`
Using Go %s features and idioms:
- Generics (if 1.18+)
- log/slog (if 1.21+)
- New routing patterns (if 1.22+)

%s
`, runtime.Version(), basePrompt)
```

### 2. Include Anti-Patterns

```go
const antiPatternPrompt = `
Avoid these Go anti-patterns:
- Empty interfaces as parameters
- Panic for error handling  
- Init functions with side effects
- Premature abstraction
- Over-use of reflection

Check for these issues in:
%s
`
```

### 3. Reference Real Examples

```go
const exampleBasedPrompt = `
Follow these excellent Go examples:
- Error handling: Look at os.Open() implementation
- Interfaces: Study io.Reader/Writer
- Concurrency: Review sync.WaitGroup usage
- HTTP: See net/http DefaultServeMux

Apply these patterns to:
%s
`
```

## 🚀 Future Enhancements

### 1. Multi-Model Prompting

```go
type MultiModelPrompt struct {
    ClaudePrompt  string // Optimized for Claude's strengths
    GeminiPrompt  string // Optimized for Gemini's strengths
    FallbackPrompt string // Generic version
}
```

### 2. Prompt Chains

```go
type PromptChain struct {
    steps []PromptStep
}

type PromptStep struct {
    Prompt      string
    Validator   func(response string) error
    NextPrompt  func(response string) string
}

// Example: Progressive code improvement
chain := PromptChain{
    steps: []PromptStep{
        {Prompt: "Identify issues", NextPrompt: generateFixPrompt},
        {Prompt: "Apply fixes", NextPrompt: generateTestPrompt},
        {Prompt: "Add tests", NextPrompt: generateDocPrompt},
    },
}
```

## 📝 Conclusion

By embedding Go best practices and CLAUDE-ARCHITECTURE principles directly into our prompts, Assistant becomes more than just an AI tool - it becomes a knowledgeable Go mentor that helps developers write better, more idiomatic code.

The key is to:
1. Always reference Go stdlib patterns
2. Enforce explicit error handling
3. Promote simple, clear solutions
4. Learn from user preferences
5. Adapt based on project context

This approach ensures that every interaction reinforces good Go practices while solving real problems efficiently.