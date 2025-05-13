package toolx

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Abraxas-365/craftable/ai/llm"
)

type Toolx interface {
	Call(ctx context.Context, inputs string) (any, error)
	GetTool() llm.Tool
	Name() string
}

type ToolxClient struct {
	tools map[string]Toolx
}

func FromToolx(tools ...Toolx) *ToolxClient {
	toolMap := make(map[string]Toolx)
	for _, tool := range tools {
		toolMap[tool.Name()] = tool
	}
	return &ToolxClient{tools: toolMap}
}

func (t *ToolxClient) GetTools() []llm.Tool {
	tools := make([]llm.Tool, 0, len(t.tools))
	for _, tool := range t.tools {
		tools = append(tools, tool.GetTool())
	}
	return tools
}

func (t *ToolxClient) Call(ctx context.Context, tc llm.ToolCall) (llm.Message, error) {
	tool, ok := t.tools[tc.Function.Name]
	if !ok {
		return llm.NewToolMessage(tc.ID, "This tool dont exists"), nil // create custom errors for this
	}

	result, err := tool.Call(ctx, tc.Function.Arguments)
	if err != nil {
		return llm.NewToolMessage(tc.ID, "Error calling tool: "+err.Error()), nil //create a custom error for this
	}

	var resultStr string
	switch v := result.(type) {
	case string:
		resultStr = v
	case []byte:
		resultStr = string(v)
	case int:
		resultStr = strconv.Itoa(v)
	case float64:
		resultStr = strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		resultStr = strconv.FormatBool(v)
	case fmt.Stringer:
		resultStr = v.String()
	default:
		// Use JSON marshaling for complex types
		jsonBytes, jsonErr := json.Marshal(result)
		if jsonErr != nil {
			return llm.NewToolMessage(tc.ID, "Error converting result to string: "+jsonErr.Error()), nil //create a custom error for this
		}
		resultStr = string(jsonBytes)
	}
	return llm.NewToolMessage(tc.ID, resultStr), nil
}
