package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

type mcpInitResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      mcpServerInfo  `json:"serverInfo"`
}

type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type mcpToolsListResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type mcpCallToolResult struct {
	Content []mcpTextContent `json:"content"`
}

func mcpLog(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[MCP-Permissions] "+format+"\n", args...)
}

func mcpReadMessage(reader *bufio.Reader) ([]byte, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(line), nil
}

func mcpWriteMessage(w io.Writer, data []byte) {
	w.Write(data)
	w.Write([]byte("\n"))
}

func mcpRespond(w io.Writer, id any, result any) {
	resp := mcpResponse{JSONRPC: "2.0", ID: id, Result: result}
	data, _ := json.Marshal(resp)
	mcpWriteMessage(w, data)
}

func mcpRespondError(w io.Writer, id any, code int, message string) {
	resp := mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   map[string]any{"code": code, "message": message},
	}
	data, _ := json.Marshal(resp)
	mcpWriteMessage(w, data)
}

func mcpHTTPPost(backendURL, path string, body any) (map[string]any, error) {
	data, _ := json.Marshal(body)
	resp, err := http.Post(backendURL+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

func mcpHTTPGet(backendURL, path string) (map[string]any, error) {
	client := &http.Client{Timeout: 70 * time.Second}
	resp, err := client.Get(backendURL + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

func mcpWaitForDecision(backendURL, requestID string) (map[string]any, error) {
	for {
		result, err := mcpHTTPGet(backendURL, "/api/permissions/"+requestID)
		if err != nil {
			return nil, err
		}
		if result["status"] != "pending" {
			return result, nil
		}
	}
}

func mcpHandleApproval(backendURL, mcpClientID string, args map[string]any) mcpCallToolResult {
	toolName, _ := args["tool_name"].(string)
	toolInput, _ := args["input"].(map[string]any)
	if toolInput == nil {
		toolInput = map[string]any{}
	}

	mcpLog("Permission request: %s", toolName)

	resp, err := mcpHTTPPost(backendURL, "/api/permissions/notify", map[string]any{
		"clientId":  mcpClientID,
		"toolName":  toolName,
		"toolInput": toolInput,
	})
	if err != nil {
		mcpLog("Error notifying backend: %v", err)
		return mcpErrorResult("Permission server error: " + err.Error())
	}

	requestID, _ := resp["requestId"].(string)
	mcpLog("Waiting for decision on %s...", requestID)

	decision, err := mcpWaitForDecision(backendURL, requestID)
	if err != nil {
		mcpLog("Error waiting for decision: %v", err)
		return mcpErrorResult("Permission server error: " + err.Error())
	}

	action, _ := decision["action"].(string)
	mcpLog("Decision: %s for %s", action, toolName)

	if action == "approve" {
		result := map[string]any{"behavior": "allow"}
		if updated, ok := decision["updatedInput"]; ok && updated != nil {
			result["updatedInput"] = updated
		} else {
			result["updatedInput"] = toolInput
		}
		data, _ := json.Marshal(result)
		return mcpCallToolResult{Content: []mcpTextContent{{Type: "text", Text: string(data)}}}
	}

	reason, _ := decision["reason"].(string)
	if reason == "" {
		reason = "User denied this action"
	}
	result := map[string]any{"behavior": "deny", "message": reason}
	data, _ := json.Marshal(result)
	return mcpCallToolResult{Content: []mcpTextContent{{Type: "text", Text: string(data)}}}
}

func mcpErrorResult(msg string) mcpCallToolResult {
	data, _ := json.Marshal(map[string]any{"behavior": "deny", "message": msg})
	return mcpCallToolResult{Content: []mcpTextContent{{Type: "text", Text: string(data)}}}
}

func runMCPServer() {
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		port := readPortFile()
		backendURL = "http://localhost:" + port
	}
	mcpClientID := os.Getenv("CLIENT_ID")

	mcpLog("Starting with BACKEND_URL=%s CLIENT_ID=%s", backendURL, mcpClientID)

	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

	for {
		body, err := mcpReadMessage(reader)
		if err != nil {
			if err == io.EOF {
				mcpLog("stdin closed, exiting")
				return
			}
			mcpLog("Read error: %v", err)
			return
		}

		var req mcpRequest
		if err := json.Unmarshal(body, &req); err != nil {
			mcpLog("Parse error: %v", err)
			continue
		}

		switch req.Method {
		case "initialize":
			mcpRespond(writer, req.ID, mcpInitResult{
				ProtocolVersion: "2024-11-05",
				Capabilities:    map[string]any{"tools": map[string]any{}},
				ServerInfo:      mcpServerInfo{Name: "clauductor-mcp", Version: "1.0.0"},
			})

		case "notifications/initialized":

		case "tools/list":
			mcpRespond(writer, req.ID, mcpToolsListResult{
				Tools: []mcpTool{{
					Name:        "approval_prompt",
					Description: "Request user approval for tool usage via web UI",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"tool_name": map[string]any{"type": "string", "title": "Tool Name"},
							"input":     map[string]any{"type": "object", "title": "Input", "additionalProperties": true},
						},
						"required": []string{"tool_name", "input"},
					},
				}},
			})

		case "tools/call":
			var params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			json.Unmarshal(req.Params, &params)

			if params.Name != "approval_prompt" {
				mcpRespondError(writer, req.ID, -32602, "Unknown tool: "+params.Name)
				continue
			}
			result := mcpHandleApproval(backendURL, mcpClientID, params.Arguments)
			mcpRespond(writer, req.ID, result)

		case "ping":
			mcpRespond(writer, req.ID, map[string]any{})

		default:
			if req.ID != nil {
				mcpRespondError(writer, req.ID, -32601, "Method not found: "+req.Method)
			}
		}
	}
}
