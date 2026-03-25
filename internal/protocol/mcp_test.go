package protocol

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPClient_ListTools(t *testing.T) {
	// 模拟 MCP 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"tools": []map[string]interface{}{
					{"name": "test_tool", "description": "A test tool"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewMCPClient(server.URL)
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(tools) != 1 || tools[0].Name != "test_tool" {
		t.Errorf("Unexpected tools result: %+v", tools)
	}
}

func TestMCPClient_CallTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "success"},
				},
				"isError": false,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewMCPClient(server.URL)
	result, err := client.CallTool(context.Background(), "test_tool", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result.IsError || len(result.Content) != 1 || result.Content[0].Text != "success" {
		t.Errorf("Unexpected CallTool result: %+v", result)
	}
}

func TestMCPClient_ErrorResp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewMCPClient(server.URL)
	err := client.Ping(context.Background())
	if err == nil {
		t.Errorf("Expected error for Ping, got nil")
	}
}
