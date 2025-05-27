package guardrailx

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Abraxas-365/craftable/ai/llm"
)

// RuleType defines the type of rule
type RuleType string

const (
	// Rule types
	BlocklistRule   RuleType = "blocklist"
	AllowlistRule   RuleType = "allowlist"
	RegexRule       RuleType = "regex"
	LengthRule      RuleType = "length"
	SensitivityRule RuleType = "sensitivity"
	CustomRule      RuleType = "custom"
	LLMRule         RuleType = "llm"
)

// ActionType defines what happens when a rule is triggered
type ActionType string

const (
	// Action types
	BlockAction  ActionType = "block"
	FilterAction ActionType = "filter"
	WarnAction   ActionType = "warn"
	LogAction    ActionType = "log"
	ModifyAction ActionType = "modify"
)

// Direction defines whether a rule applies to input or output
type Direction string

const (
	Input  Direction = "input"
	Output Direction = "output"
	Both   Direction = "both"
)

// LLMCheckConfig holds configuration for LLM-based checking
type LLMCheckConfig struct {
	Client        *llm.Client
	InputPrompt   string
	OutputPrompt  string
	ModelOptions  []llm.Option
	ValidResponse []string
}

// Rule defines a validation rule for messages
type Rule struct {
	Name        string
	Type        RuleType
	Pattern     any
	Direction   Direction
	Action      ActionType
	Replacement string
	Message     string
	Priority    int
	CustomCheck RuleCheckFunc
	LLMConfig   *LLMCheckConfig
}

// RuleCheckFunc is a custom function to check if a rule is violated
type RuleCheckFunc func(input string) (bool, string, error)

// ValidationResult represents the outcome of validating a message
type ValidationResult struct {
	IsValid    bool
	Modified   bool
	Message    string
	Rule       *Rule
	Violations []string
	Details    string
}

// Logger interface for logging violations
type Logger interface {
	Log(rule Rule, content string, direction Direction, result ValidationResult) error
}

// DefaultLogger provides basic logging capabilities
type DefaultLogger struct{}

func (l *DefaultLogger) Log(rule Rule, content string, direction Direction, result ValidationResult) error {
	fmt.Printf("[%s] Rule '%s' triggered for %s: %s\n", rule.Type, rule.Name, direction, result.Message)
	if result.Details != "" {
		fmt.Printf("  Details: %s\n", result.Details)
	}
	return nil
}

// GuardRailOption configures a GuardRail
type GuardRailOption func(*GuardRail)

// WithLogger sets a custom logger
func WithLogger(logger Logger) GuardRailOption {
	return func(g *GuardRail) {
		g.logger = logger
	}
}

// WithViolationHandler sets a callback for rule violations
func WithViolationHandler(handler func(Rule, string, Direction) error) GuardRailOption {
	return func(g *GuardRail) {
		g.onViolation = handler
	}
}

// WithDefaultLLM sets a default LLM client for all LLM rules that don't specify one
func WithDefaultLLM(client *llm.Client, options []llm.Option) GuardRailOption {
	return func(g *GuardRail) {
		g.defaultLLMClient = client
		g.defaultLLMOptions = options
	}
}

// GuardRail manages rules and validates messages
type GuardRail struct {
	rules             []Rule
	logger            Logger
	onViolation       func(Rule, string, Direction) error
	defaultLLMClient  *llm.Client
	defaultLLMOptions []llm.Option
}

// NewGuardRail creates a new guard rail with the given rules
func NewGuardRail(rules []Rule, opts ...GuardRailOption) *GuardRail {
	g := &GuardRail{
		rules:  rules,
		logger: &DefaultLogger{},
		onViolation: func(r Rule, content string, d Direction) error {
			return nil
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(g)
	}

	// Sort rules by priority
	sortRulesByPriority(g.rules)

	return g
}

// sortRulesByPriority sorts rules by priority (higher first)
func sortRulesByPriority(rules []Rule) {
	for i := 1; i < len(rules); i++ {
		j := i
		for j > 0 && rules[j-1].Priority < rules[j].Priority {
			rules[j], rules[j-1] = rules[j-1], rules[j]
			j--
		}
	}
}

// ValidateInput checks if input text passes all input rules
func (g *GuardRail) ValidateInput(ctx context.Context, input string) (ValidationResult, error) {
	return g.validate(ctx, input, Input)
}

// ValidateOutput checks if output text passes all output rules
func (g *GuardRail) ValidateOutput(ctx context.Context, output string) (ValidationResult, error) {
	return g.validate(ctx, output, Output)
}

// validate checks a message against applicable rules
func (g *GuardRail) validate(ctx context.Context, content string, direction Direction) (ValidationResult, error) {
	result := ValidationResult{
		IsValid:    true,
		Modified:   false,
		Message:    content,
		Violations: []string{},
	}

	// Check each rule that applies to this direction
	for i := range g.rules {
		rule := &g.rules[i]
		if rule.Direction != direction && rule.Direction != Both {
			continue
		}

		var violated bool
		var replacedContent string
		var err error
		var details string

		// For LLM rules, we need to pass the context
		if rule.Type == LLMRule {
			violated, replacedContent, details, err = g.checkLLM(ctx, *rule, content, direction)
		} else {
			violated, replacedContent, err = g.checkRule(*rule, content)
		}

		if err != nil {
			return result, fmt.Errorf("error checking rule %s: %w", rule.Name, err)
		}

		if violated {
			// Log the violation
			vResult := ValidationResult{
				IsValid:    false,
				Modified:   rule.Action == FilterAction || rule.Action == ModifyAction,
				Message:    rule.Message,
				Rule:       rule,
				Violations: []string{rule.Name},
				Details:    details,
			}
			g.logger.Log(*rule, content, direction, vResult)

			// Call the violation handler
			if g.onViolation != nil {
				if err := g.onViolation(*rule, content, direction); err != nil {
					return result, err
				}
			}

			// Handle the violation according to the rule's action
			result.Violations = append(result.Violations, rule.Name)
			result.Details = details

			switch rule.Action {
			case BlockAction:
				return ValidationResult{
					IsValid:    false,
					Modified:   false,
					Message:    rule.Message,
					Rule:       rule,
					Violations: result.Violations,
					Details:    details,
				}, nil

			case FilterAction:
				result.Modified = true
				result.Message = replacedContent
				content = replacedContent

			case WarnAction:
				// Continue with validation but track the warning
				result.IsValid = true

			case LogAction:
				// Just log, no further action

			case ModifyAction:
				result.Modified = true
				result.Message = replacedContent
				content = replacedContent
			}
		}
	}

	// If we've modified the content, update the result
	if result.Modified {
		result.Message = content
	}

	return result, nil
}

// checkRule determines if a specific rule is violated
func (g *GuardRail) checkRule(rule Rule, content string) (bool, string, error) {
	switch rule.Type {
	case BlocklistRule:
		return g.checkBlocklist(rule, content)
	case AllowlistRule:
		return g.checkAllowlist(rule, content)
	case RegexRule:
		return g.checkRegex(rule, content)
	case LengthRule:
		return g.checkLength(rule, content)
	case SensitivityRule:
		return g.checkSensitivity(rule, content)
	case CustomRule:
		return g.checkCustom(rule, content)
	default:
		return false, content, fmt.Errorf("unknown rule type: %s", rule.Type)
	}
}

// checkLLM uses an LLM to evaluate content
func (g *GuardRail) checkLLM(ctx context.Context, rule Rule, content string, direction Direction) (bool, string, string, error) {
	if rule.LLMConfig == nil {
		return false, content, "", fmt.Errorf("LLM rule requires LLMConfig")
	}

	config := rule.LLMConfig

	// Use default LLM client if not specified in the rule
	client := config.Client
	if client == nil {
		if g.defaultLLMClient == nil {
			return false, content, "", fmt.Errorf("no LLM client specified for rule %s and no default client set", rule.Name)
		}
		client = g.defaultLLMClient
	}

	// Select the appropriate prompt based on direction
	var promptTemplate string
	switch direction {
	case Input:
		promptTemplate = config.InputPrompt
	case Output:
		promptTemplate = config.OutputPrompt
	default:
		return false, content, "", fmt.Errorf("invalid direction for LLM rule: %s", direction)
	}

	if promptTemplate == "" {
		return false, content, "", fmt.Errorf("no prompt template specified for direction %s", direction)
	}

	// Replace {{content}} placeholder with actual content
	prompt := strings.ReplaceAll(promptTemplate, "{{content}}", content)

	// Prepare messages for the LLM
	messages := []llm.Message{
		llm.NewSystemMessage("You are a content moderation assistant. Evaluate the content and respond with ONLY 'yes' or 'no'."),
		llm.NewUserMessage(prompt),
	}

	// Combine rule-specific options with default options
	options := config.ModelOptions
	if len(options) == 0 {
		options = g.defaultLLMOptions
	}

	// Call the LLM
	response, err := client.Chat(ctx, messages, options...)
	if err != nil {
		return false, content, "", fmt.Errorf("LLM error: %w", err)
	}

	// Process the response
	llmResponse := strings.TrimSpace(response.Message.Content)

	// Debug
	// fmt.Printf("LLM Response for rule '%s': '%s'\n", rule.Name, llmResponse)

	// Check if the response indicates a violation
	violated := false
	if len(config.ValidResponse) > 0 {
		// User specified what responses are valid
		valid := false
		for _, validResp := range config.ValidResponse {
			if strings.EqualFold(llmResponse, validResp) {
				valid = true
				break
			}
		}
		violated = !valid
	} else {
		// Default: assume 'no' means violated
		violated = !strings.EqualFold(llmResponse, "yes")
	}

	return violated, content, llmResponse, nil
}

// checkBlocklist checks if any blocklisted terms are present
func (g *GuardRail) checkBlocklist(rule Rule, content string) (bool, string, error) {
	terms, ok := rule.Pattern.([]string)
	if !ok {
		return false, content, fmt.Errorf("blocklist pattern must be a string slice")
	}

	lowerContent := strings.ToLower(content)
	replacedContent := content

	for _, term := range terms {
		if strings.Contains(lowerContent, strings.ToLower(term)) {
			if rule.Action == FilterAction && rule.Replacement != "" {
				// Replace all instances of the term
				re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(term))
				if err != nil {
					return true, content, err
				}
				replacedContent = re.ReplaceAllString(replacedContent, rule.Replacement)
			}
			return true, replacedContent, nil
		}
	}

	return false, content, nil
}

// checkAllowlist checks if only allowlisted terms are present
func (g *GuardRail) checkAllowlist(rule Rule, content string) (bool, string, error) {
	patterns, ok := rule.Pattern.([]string)
	if !ok {
		return false, content, fmt.Errorf("allowlist pattern must be a string slice")
	}

	// Create a mega-pattern from all allowed patterns
	allAllowed := strings.Join(patterns, "|")
	re, err := regexp.Compile("(?i)(" + allAllowed + ")")
	if err != nil {
		return false, content, err
	}

	// Replace all allowed content with empty space to see what's left
	sanitized := re.ReplaceAllString(content, "")
	sanitized = strings.TrimSpace(sanitized)

	// If anything is left after removing allowed content, it's a violation
	if sanitized != "" {
		return true, content, nil
	}

	return false, content, nil
}

// checkRegex checks if content matches a regex pattern
func (g *GuardRail) checkRegex(rule Rule, content string) (bool, string, error) {
	pattern, ok := rule.Pattern.(string)
	if !ok {
		return false, content, fmt.Errorf("regex pattern must be a string")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, content, fmt.Errorf("invalid regex pattern: %w", err)
	}

	matches := re.FindAllString(content, -1)
	if len(matches) > 0 {
		if rule.Action == FilterAction && rule.Replacement != "" {
			// Replace all matches
			replacedContent := re.ReplaceAllString(content, rule.Replacement)
			return true, replacedContent, nil
		}
		return true, content, nil
	}

	return false, content, nil
}

// checkLength checks if content length is within bounds
func (g *GuardRail) checkLength(rule Rule, content string) (bool, string, error) {
	limits, ok := rule.Pattern.(map[string]int)
	if !ok {
		return false, content, fmt.Errorf("length pattern must be a map[string]int")
	}

	length := len(content)

	// Check min length
	if min, exists := limits["min"]; exists && length < min {
		return true, content, nil
	}

	// Check max length
	if max, exists := limits["max"]; exists && length > max {
		// If it's too long and we're filtering, truncate it
		if rule.Action == FilterAction || rule.Action == ModifyAction {
			if length > max {
				return true, content[:max] + "...", nil
			}
		}
		return true, content, nil
	}

	return false, content, nil
}

// checkSensitivity checks for sensitive information patterns
func (g *GuardRail) checkSensitivity(rule Rule, content string) (bool, string, error) {
	patterns, ok := rule.Pattern.(map[string]string)
	if !ok {
		return false, content, fmt.Errorf("sensitivity pattern must be a map[string]string")
	}

	replacedContent := content
	violated := false

	for name, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, content, fmt.Errorf("invalid pattern for %s: %w", name, err)
		}

		if re.MatchString(content) {
			violated = true
			if rule.Action == FilterAction && rule.Replacement != "" {
				replacedContent = re.ReplaceAllString(replacedContent, rule.Replacement)
			}
		}
	}

	return violated, replacedContent, nil
}

// checkCustom uses a custom function to validate content
func (g *GuardRail) checkCustom(rule Rule, content string) (bool, string, error) {
	if rule.CustomCheck == nil {
		return false, content, fmt.Errorf("custom rule requires a CustomCheck function")
	}

	return rule.CustomCheck(content)
}

// FallbackHandler handles cases where guardrails block a message
type FallbackHandler interface {
	HandleBlockedInput(ctx context.Context, input string, result ValidationResult) (string, error)
	HandleBlockedOutput(ctx context.Context, output string, result ValidationResult) (string, error)
}

// DefaultFallbackHandler provides standard fallback responses
type DefaultFallbackHandler struct{}

func (h *DefaultFallbackHandler) HandleBlockedInput(ctx context.Context, input string, result ValidationResult) (string, error) {
	if result.Rule != nil && result.Rule.Message != "" {
		return result.Rule.Message, nil
	}
	return "I'm unable to process that request due to content policy restrictions.", nil
}

func (h *DefaultFallbackHandler) HandleBlockedOutput(ctx context.Context, output string, result ValidationResult) (string, error) {
	if result.Rule != nil && result.Rule.Message != "" {
		return result.Rule.Message, nil
	}
	return "I apologize, but I'm unable to provide that information due to content policy restrictions.", nil
}

// GuardedAgent wraps an Agent with guardrails
type GuardedAgent struct {
	agent interface {
		Run(ctx context.Context, userInput string) (string, error)
	}
	guardrail *GuardRail
	fallback  FallbackHandler
}

// NewGuardedAgent creates a new agent with guardrails
func NewGuardedAgent(agent interface {
	Run(ctx context.Context, userInput string) (string, error)
}, guardrail *GuardRail, fallback FallbackHandler) *GuardedAgent {
	if fallback == nil {
		fallback = &DefaultFallbackHandler{}
	}

	return &GuardedAgent{
		agent:     agent,
		guardrail: guardrail,
		fallback:  fallback,
	}
}

// Run processes input through guardrails, runs the agent, and validates the output
func (a *GuardedAgent) Run(ctx context.Context, userInput string) (string, error) {
	// Skip empty inputs
	if strings.TrimSpace(userInput) == "" {
		return "", nil
	}

	// Validate input
	inputResult, err := a.guardrail.ValidateInput(ctx, userInput)
	if err != nil {
		return "", fmt.Errorf("input validation error: %w", err)
	}

	// If input is invalid and blocked
	if !inputResult.IsValid && inputResult.Rule != nil && inputResult.Rule.Action == BlockAction {
		return a.fallback.HandleBlockedInput(ctx, userInput, inputResult)
	}

	// Use modified input if applicable
	processedInput := userInput
	if inputResult.Modified {
		processedInput = inputResult.Message
	}

	// Get response from agent
	response, err := a.agent.Run(ctx, processedInput)
	if err != nil {
		return "", err
	}

	// Skip validation for empty responses
	if strings.TrimSpace(response) == "" {
		return response, nil
	}

	// Validate output
	outputResult, err := a.guardrail.ValidateOutput(ctx, response)
	if err != nil {
		return "", fmt.Errorf("output validation error: %w", err)
	}

	// If output is invalid and blocked
	if !outputResult.IsValid && outputResult.Rule != nil && outputResult.Rule.Action == BlockAction {
		return a.fallback.HandleBlockedOutput(ctx, response, outputResult)
	}

	// Return modified output if applicable
	if outputResult.Modified {
		return outputResult.Message, nil
	}

	return response, nil
}
