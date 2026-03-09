package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var runningProcesses = struct {
	sync.Mutex
	m map[string]*exec.Cmd
}{m: make(map[string]*exec.Cmd)}

var sessionRunClients = struct {
	sync.RWMutex
	m map[string]string
}{m: make(map[string]string)}

type SessionMeta struct {
	SessionID string    `json:"sessionId"`
	CWD       string    `json:"cwd"`
	StartedAt time.Time `json:"startedAt"`
}

var sessionMetaStore = struct {
	sync.RWMutex
	m map[string]SessionMeta
}{m: make(map[string]SessionMeta)}

type Profile struct {
	Name string            `json:"name"`
	Env  map[string]string `json:"env"`
}

type ProfilesConfig struct {
	Active   string             `json:"active"`
	Profiles map[string]Profile `json:"profiles"`
}

func profilesPath() string {
	return filepath.Join(clauductorDir(), "profiles.json")
}

func loadProfiles() ProfilesConfig {
	data, err := os.ReadFile(profilesPath())
	if err != nil {
		cfg := ProfilesConfig{
			Active: "default",
			Profiles: map[string]Profile{
				"default": {
					Name: "Default",
					Env:  map[string]string{},
				},
			},
		}
		saveProfiles(cfg)
		return cfg
	}

	var cfg ProfilesConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("[Profiles] Failed to parse profiles.json: %v", err)
		return ProfilesConfig{Active: "default", Profiles: map[string]Profile{
			"default": {Name: "Default", Env: map[string]string{}},
		}}
	}
	return cfg
}

func saveProfiles(cfg ProfilesConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(profilesPath(), data, 0644)
}

func getProfilesHandler(c *gin.Context) {
	cfg := loadProfiles()
	c.JSON(http.StatusOK, cfg)
}

func saveProfilesHandler(c *gin.Context) {
	var cfg ProfilesConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := saveProfiles(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

func setActiveProfileHandler(c *gin.Context) {
	var body struct {
		Active string `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg := loadProfiles()
	if _, ok := cfg.Profiles[body.Active]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile not found"})
		return
	}
	cfg.Active = body.Active
	if err := saveProfiles(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "active": cfg.Active})
}

type PermissionReq struct {
	ID           string         `json:"id"`
	ClientID     string         `json:"clientId"`
	ToolName     string         `json:"toolName"`
	ToolInput    map[string]any `json:"toolInput"`
	Status       string         `json:"status"`
	Action       string         `json:"action"`
	Reason       string         `json:"reason,omitempty"`
	UpdatedInput map[string]any `json:"updatedInput,omitempty"`
	decided      chan struct{}
}

var permissionStore = struct {
	sync.Mutex
	m map[string]*PermissionReq
}{m: make(map[string]*PermissionReq)}

type RunRequest struct {
	Prompt         string `json:"prompt"`
	Cwd            string `json:"cwd"`
	ConversationID string `json:"conversationId"`
	Model           string `json:"model"`
	Mode            string `json:"mode"`
	PermissionStyle string `json:"permissionStyle"`
	ClientID        string `json:"clientId"`
}

func findBinary(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	home, _ := os.UserHomeDir()
	knownPaths := []string{
		home + "/.claude/local/" + name,
		home + "/.local/bin/" + name,
		"/usr/local/bin/" + name,
	}
	for _, p := range knownPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	out, err := exec.Command(shell, "-ilc", "which "+name).Output()
	if err == nil {
		if path := strings.TrimSpace(string(out)); path != "" {
			return path, nil
		}
	}

	return "", fmt.Errorf("%s not found", name)
}

func runHandler(c *gin.Context) {
	var req RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clientId required"})
		return
	}

	go executeClaude(req)
	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

func executeClaude(req RunRequest) {
	log.Printf("[Claude] prompt=%q cwd=%q convID=%q model=%q mode=%q client=%s",
		req.Prompt, req.Cwd, req.ConversationID, req.Model, req.Mode, req.ClientID)

	activeSessionId := req.ConversationID
	if activeSessionId != "" {
		sessionRunClients.Lock()
		sessionRunClients.m[activeSessionId] = req.ClientID
		sessionRunClients.Unlock()
		sessionMetaStore.Lock()
		sessionMetaStore.m[activeSessionId] = SessionMeta{SessionID: activeSessionId, CWD: req.Cwd, StartedAt: time.Now()}
		sessionMetaStore.Unlock()
	}

	sendMsg := func(msg Message) {
		targetClient := req.ClientID
		if activeSessionId != "" {
			sessionRunClients.RLock()
			if cid, ok := sessionRunClients.m[activeSessionId]; ok {
				targetClient = cid
			}
			sessionRunClients.RUnlock()
		}
		manager.Send(targetClient, msg)
	}

	profilesCfg := loadProfiles()
	activeProfile, profileOk := profilesCfg.Profiles[profilesCfg.Active]
	if !profileOk {
		activeProfile = Profile{Env: map[string]string{}}
	}
	log.Printf("[Claude] Using profile: %s", profilesCfg.Active)

	model := req.Model
	if model == "" {
		model = "sonnet"
	}

	command, err := findBinary("claude")
	if err != nil {
		log.Printf("[Claude] claude binary not found: %v", err)
		sendMsg(Message{Type: "error", Data: "claude binary not found in PATH"})
		return
	}
	log.Printf("[Claude] Using binary: %s", command)

	yolo := req.PermissionStyle == "yolo"

	args := []string{
		"-p", req.Prompt,
		"--model", model,
		"--output-format", "stream-json",
		"--verbose",
	}

	if yolo {
		args = append(args, "--dangerously-skip-permissions")
	} else {
		args = append(args, "--permission-prompt-tool", "mcp__clauductor-mcp__approval_prompt")
	}

	if req.Mode == "plan" {
		args = append(args, "--permission-mode", "plan")
	}

	if req.ConversationID != "" {
		args = append(args, "-c", req.ConversationID)
	}

	log.Printf("[Claude] Exec: %s %s", command, strings.Join(args, " "))

	cmd := exec.Command(command, args...)
	if req.Cwd != "" {
		cmd.Dir = req.Cwd
	}

	var filteredEnv []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "CLAUDECODE=") {
			filteredEnv = append(filteredEnv, e)
		}
	}
	cmd.Env = append(filteredEnv, "CLIENT_ID="+req.ClientID)
	for k, v := range activeProfile.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[Claude] StdoutPipe error: %v", err)
		manager.Send(req.ClientID, Message{Type: "error", Data: err.Error()})
		return
	}

	stderrPipe, _ := cmd.StderrPipe()
	var stderrBuf bytes.Buffer

	if err := cmd.Start(); err != nil {
		log.Printf("[Claude] Start error: %v", err)
		sendMsg(Message{Type: "error", Data: err.Error()})
		return
	}

	go func() {
		stderrScanner := bufio.NewScanner(stderrPipe)
		stderrScanner.Buffer(make([]byte, 0, 64*1024), 64*1024)
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			stderrBuf.WriteString(line + "\n")
			log.Printf("[Claude:stderr] %s", line)
			sendMsg(Message{Type: "stderr", Data: line})
		}
	}()

	runningProcesses.Lock()
	runningProcesses.m[req.ClientID] = cmd
	runningProcesses.Unlock()
	defer func() {
		runningProcesses.Lock()
		delete(runningProcesses.m, req.ClientID)
		runningProcesses.Unlock()

		if activeSessionId != "" {
			sessionRunClients.Lock()
			delete(sessionRunClients.m, activeSessionId)
			sessionRunClients.Unlock()
			sessionMetaStore.Lock()
			delete(sessionMetaStore.m, activeSessionId)
			sessionMetaStore.Unlock()
		}

		permissionStore.Lock()
		for id, p := range permissionStore.m {
			if p.ClientID == req.ClientID && p.Status == "pending" {
				p.Status = "denied"
				p.Action = "deny"
				p.Reason = "Process ended"
				close(p.decided)
				delete(permissionStore.m, id)
			}
		}
		permissionStore.Unlock()
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		var parsed map[string]any
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			sendMsg(Message{Type: "log", Data: line})
			continue
		}

		if activeSessionId == "" {
			if t, _ := parsed["type"].(string); t == "system" {
				if st, _ := parsed["subtype"].(string); st == "init" {
					if sid, _ := parsed["session_id"].(string); sid != "" {
						activeSessionId = sid
						sessionRunClients.Lock()
						sessionRunClients.m[activeSessionId] = req.ClientID
						sessionRunClients.Unlock()
						sessionMetaStore.Lock()
						sessionMetaStore.m[activeSessionId] = SessionMeta{SessionID: activeSessionId, CWD: req.Cwd, StartedAt: time.Now()}
						sessionMetaStore.Unlock()
						log.Printf("[Claude] Session registered: %s -> client %s", activeSessionId, req.ClientID)
					}
				}
			}
		}

		sendMsg(Message{Type: "claude_event", Data: parsed})
	}

	if err := cmd.Wait(); err != nil {
		stderrOut := stderrBuf.String()
		log.Printf("[Claude] Process exited: %v", err)
		if stderrOut != "" {
			log.Printf("[Claude] Stderr: %s", stderrOut)
		}
	}

	sendMsg(Message{Type: "done"})
}

func permissionNotifyHandler(c *gin.Context) {
	var body struct {
		ClientID  string         `json:"clientId"`
		ToolName  string         `json:"toolName"`
		ToolInput map[string]any `json:"toolInput"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	b := make([]byte, 16)
	rand.Read(b)
	reqID := fmt.Sprintf("%x", b)
	perm := &PermissionReq{
		ID:        reqID,
		ClientID:  body.ClientID,
		ToolName:  body.ToolName,
		ToolInput: body.ToolInput,
		Status:    "pending",
		decided:   make(chan struct{}),
	}

	permissionStore.Lock()
	permissionStore.m[reqID] = perm
	permissionStore.Unlock()

	log.Printf("[Permission] New request %s: tool=%s client=%s", reqID, body.ToolName, body.ClientID)

	manager.Send(body.ClientID, Message{
		Type: "permission_request",
		Data: map[string]any{
			"requestId": reqID,
			"toolName":  body.ToolName,
			"toolInput": body.ToolInput,
		},
	})

	c.JSON(http.StatusOK, gin.H{"requestId": reqID})
}

func permissionPollHandler(c *gin.Context) {
	id := c.Param("id")

	permissionStore.Lock()
	perm, ok := permissionStore.m[id]
	permissionStore.Unlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	select {
	case <-perm.decided:
		c.JSON(http.StatusOK, gin.H{
			"status":       perm.Status,
			"action":       perm.Action,
			"reason":       perm.Reason,
			"updatedInput": perm.UpdatedInput,
		})
		permissionStore.Lock()
		delete(permissionStore.m, id)
		permissionStore.Unlock()

	case <-time.After(60 * time.Second):
		c.JSON(http.StatusOK, gin.H{"status": "pending"})

	case <-c.Request.Context().Done():
		return
	}
}

func permissionDecideHandler(c *gin.Context) {
	id := c.Param("id")

	var body struct {
		Action       string         `json:"action"`
		Reason       string         `json:"reason"`
		UpdatedInput map[string]any `json:"updatedInput"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	permissionStore.Lock()
	perm, ok := permissionStore.m[id]
	permissionStore.Unlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	perm.Status = body.Action + "d"
	perm.Action = body.Action
	perm.Reason = body.Reason
	perm.UpdatedInput = body.UpdatedInput
	close(perm.decided)

	log.Printf("[Permission] Decision for %s: action=%s status=%s updatedInput=%v", id, body.Action, perm.Status, body.UpdatedInput)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func stopHandler(c *gin.Context) {
	var body struct {
		ClientID string `json:"clientId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clientId required"})
		return
	}

	runningProcesses.Lock()
	cmd, ok := runningProcesses.m[body.ClientID]
	runningProcesses.Unlock()

	if !ok || cmd.Process == nil {
		c.JSON(http.StatusOK, gin.H{"status": "no process"})
		return
	}

	log.Printf("[Stop] Sending SIGINT to process %d for client %s", cmd.Process.Pid, body.ClientID)
	cmd.Process.Signal(os.Interrupt)
	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

func checkPathHandler(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	info, err := os.Stat(req.Path)
	exists := err == nil && info.IsDir()
	c.JSON(http.StatusOK, gin.H{"exists": exists})
}

func createPathHandler(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := os.MkdirAll(req.Path, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "created"})
}

func sessionClaimHandler(c *gin.Context) {
	sessionId := c.Param("id")
	var body struct {
		ClientID string `json:"clientId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clientId required"})
		return
	}

	sessionRunClients.Lock()
	_, isActive := sessionRunClients.m[sessionId]
	if isActive {
		sessionRunClients.m[sessionId] = body.ClientID
		log.Printf("[Claude] Session %s claimed by new client %s", sessionId, body.ClientID)
	}
	sessionRunClients.Unlock()

	c.JSON(http.StatusOK, gin.H{"active": isActive})
}

func getConfigHandler(c *gin.Context) {
	path := filepath.Join(claudeDir(), "settings.json")
	data, err := os.ReadFile(path)

	defaultConfig := gin.H{
		"model":             "sonnet",
		"outputStyle":       "Standard",
		"cleanupPeriodDays": 30,
		"respectGitignore":  true,
		"permissions": gin.H{
			"allow": []string{},
			"deny":  []string{},
		},
		"env": gin.H{},
		"attribution": gin.H{
			"commit": "🤖 Generated with Claude Code",
			"pr":     "",
		},
		"includeGitInstructions": true,
	}

	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, defaultConfig)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		c.JSON(http.StatusOK, defaultConfig)
		return
	}

	c.JSON(http.StatusOK, config)
}

func saveConfigHandler(c *gin.Context) {
	var config any
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	path := filepath.Join(claudeDir(), "settings.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

type UIPrefs struct {
	Model           string `json:"model"`
	PermissionMode  string `json:"permissionMode"`
	PermissionStyle string `json:"permissionStyle"`
	Cwd             string `json:"cwd"`
	ConversationID  string `json:"conversationId"`
}

func prefsPath() string {
	return filepath.Join(clauductorDir(), "prefs.json")
}

func getPrefsHandler(c *gin.Context) {
	defaults := UIPrefs{
		Model:           "sonnet",
		PermissionMode:  "agent",
		PermissionStyle: "ask",
	}
	data, err := os.ReadFile(prefsPath())
	if err != nil {
		c.JSON(http.StatusOK, defaults)
		return
	}
	var prefs UIPrefs
	if err := json.Unmarshal(data, &prefs); err != nil {
		c.JSON(http.StatusOK, defaults)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func savePrefsHandler(c *gin.Context) {
	var prefs UIPrefs
	if err := c.ShouldBindJSON(&prefs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := os.WriteFile(prefsPath(), data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

func activeSessionsHandler(c *gin.Context) {
	sessionRunClients.RLock()
	activeIds := make([]string, 0, len(sessionRunClients.m))
	for id := range sessionRunClients.m {
		activeIds = append(activeIds, id)
	}
	sessionRunClients.RUnlock()

	sessionMetaStore.RLock()
	result := make([]SessionMeta, 0, len(activeIds))
	for _, id := range activeIds {
		if meta, ok := sessionMetaStore.m[id]; ok {
			result = append(result, meta)
		} else {
			result = append(result, SessionMeta{SessionID: id})
		}
	}
	sessionMetaStore.RUnlock()

	c.JSON(http.StatusOK, result)
}

func claudeVersionHandler(c *gin.Context) {
	out, err := exec.Command("claude", "-v").Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"version": strings.TrimSpace(string(out))})
}

func claudeUpdateHandler(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	pr, pw := io.Pipe()
	cmd := exec.Command("claude", "update")
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", err.Error())
		c.Writer.Flush()
		return
	}

	type lineMsg struct {
		line string
		err  error
		done bool
	}
	ch := make(chan lineMsg, 100)

	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			ch <- lineMsg{line: scanner.Text()}
		}
		if err := scanner.Err(); err != nil {
			ch <- lineMsg{err: err}
		} else {
			ch <- lineMsg{done: true}
		}
		close(ch)
	}()

	go func() {
		cmd.Wait()
		pw.Close()
	}()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			pr.Close()
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg.done {
				fmt.Fprintf(c.Writer, "event: done\ndata: \n\n")
				c.Writer.Flush()
				return
			}
			if msg.err != nil {
				fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", msg.err.Error())
				c.Writer.Flush()
				return
			}
			fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", msg.line)
			c.Writer.Flush()
		}
	}
}

func usageHandler(c *gin.Context) {
	token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if token == "" {
		out, err := exec.Command("security", "find-generic-password", "-a", os.Getenv("USER"), "-w", "-s", "Claude Code").Output()
		if err == nil {
			token = strings.TrimSpace(string(out))
		}
	}

	if token == "" {
		home := os.Getenv("HOME")
		credPath := filepath.Join(home, ".claude", ".credentials.json")
		data, err := os.ReadFile(credPath)
		if err == nil {
			var creds struct {
				ClaudeAiOauth struct {
					AccessToken string `json:"accessToken"`
				} `json:"claudeAiOauth"`
			}
			if err := json.Unmarshal(data, &creds); err == nil {
				token = creds.ClaudeAiOauth.AccessToken
			}
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not logged in to Claude Code. Run: claude login"})
		return
	}

	if len(token) >= 2 && token[:2] == "0x" {
		if decoded, err := hex.DecodeString(token[2:]); err == nil {
			token = string(decoded)
		}
	} else if len(token)%2 == 0 && len(token) > 20 {
		isHex := true
		for _, ch := range token {
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				isHex = false
				break
			}
		}
		if isHex {
			if decoded, err := hex.DecodeString(token); err == nil {
				token = string(decoded)
			}
		}
	}

	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://api.anthropic.com/api/oauth/usage", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "claude-code/2.0.0")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(body, &result)

	if resp.StatusCode != 200 {
		result["error"] = string(body)
	}

	c.JSON(http.StatusOK, result)
}
