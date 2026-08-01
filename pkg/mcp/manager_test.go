package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestConvertSingleToolAlwaysIncludesParameters(t *testing.T) {
	manager := &MCPManager{}
	tool, err := manager.convertSingleTool("builtin", &sdkmcp.Tool{
		Name:        "health_check",
		Description: "No-argument health check",
	})
	if err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(tool)
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(payload)
	if !strings.Contains(jsonText, `"parameters":{"properties":{},"type":"object"}`) &&
		!strings.Contains(jsonText, `"parameters":{"type":"object","properties":{}}`) {
		t.Fatalf("no-argument tool is missing an object parameters schema: %s", jsonText)
	}
}
