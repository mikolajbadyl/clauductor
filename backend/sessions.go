package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type Project struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	EncodedDir string `json:"encodedDir"`
}

type SessionSummary struct {
	ID        string `json:"id"`
	Display   string `json:"display"`
	Timestamp int64  `json:"timestamp"`
	Project   string `json:"project"`
}

type SessionMessage struct {
	UUID      string `json:"uuid"`
	Type      string `json:"type"`
	Role      string `json:"role"`
	Content   any    `json:"content"`
	Timestamp string `json:"timestamp"`
	Model     string `json:"model,omitempty"`
}

type historyEntry struct {
	Display   string `json:"display"`
	Timestamp int64  `json:"timestamp"`
	Project   string `json:"project"`
	SessionID string `json:"sessionId"`
}

type sessionJSONLEntry struct {
	Type      string `json:"type"`
	UUID      string `json:"uuid"`
	Timestamp string `json:"timestamp"`
	Message   struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
		Model   string `json:"model"`
	} `json:"message"`
}

func projectsHandler(c *gin.Context) {
	projects, err := ListProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, projects)
}

func sessionsHandler(c *gin.Context) {
	encoded := c.Param("encoded")
	sessions, err := ListSessions(encoded)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

func sessionHandler(c *gin.Context) {
	encoded := c.Param("encoded")
	id := c.Param("id")
	messages, err := LoadSession(encoded, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, messages)
}

func claudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

func encodeProjectPath(p string) string {
	return strings.ReplaceAll(p, "/", "-")
}

func ListProjects() ([]Project, error) {
	pathMap := buildProjectPathMap()

	var projects []Project
	for encoded, realPath := range pathMap {
		if info, err := os.Stat(realPath); err != nil || !info.IsDir() {
			continue
		}

		projects = append(projects, Project{
			Name:       filepath.Base(realPath),
			Path:       realPath,
			EncodedDir: encoded,
		})
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}

func ListSessions(encodedProject string) ([]SessionSummary, error) {
	historyIndex := buildHistoryIndex()

	projectDir := filepath.Join(claudeDir(), "projects", encodedProject)
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionSummary{}, nil
		}
		return nil, err
	}

	var sessions []SessionSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		sessionID := strings.TrimSuffix(e.Name(), ".jsonl")

		summary := SessionSummary{ID: sessionID}
		if h, ok := historyIndex[sessionID]; ok {
			summary.Display = h.Display
			summary.Timestamp = h.Timestamp
			summary.Project = h.Project
		} else {
			summary.Display = firstPromptFromSession(filepath.Join(projectDir, e.Name()))
		}

		sessions = append(sessions, summary)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Timestamp > sessions[j].Timestamp
	})

	return sessions, nil
}

func LoadSession(encodedProject, sessionID string) ([]SessionMessage, error) {
	path := filepath.Join(claudeDir(), "projects", encodedProject, sessionID+".jsonl")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var messages []SessionMessage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 2*1024*1024), 2*1024*1024)

	for scanner.Scan() {
		var entry sessionJSONLEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		if entry.Type != "user" && entry.Type != "assistant" {
			continue
		}

		messages = append(messages, SessionMessage{
			UUID:      entry.UUID,
			Type:      entry.Type,
			Role:      entry.Message.Role,
			Content:   entry.Message.Content,
			Timestamp: entry.Timestamp,
			Model:     entry.Message.Model,
		})
	}

	return messages, scanner.Err()
}

func buildHistoryIndex() map[string]historyEntry {
	index := make(map[string]historyEntry)
	path := filepath.Join(claudeDir(), "history.jsonl")

	f, err := os.Open(path)
	if err != nil {
		return index
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 512*1024), 512*1024)

	for scanner.Scan() {
		var entry historyEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.SessionID == "" {
			continue
		}
		if existing, ok := index[entry.SessionID]; !ok || entry.Timestamp < existing.Timestamp {
			index[entry.SessionID] = entry
		}
	}

	return index
}

func buildProjectPathMap() map[string]string {
	m := make(map[string]string)
	path := filepath.Join(claudeDir(), "history.jsonl")

	f, err := os.Open(path)
	if err != nil {
		return m
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 512*1024), 512*1024)

	for scanner.Scan() {
		var entry historyEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil || entry.Project == "" {
			continue
		}
		encoded := encodeProjectPath(entry.Project)
		m[encoded] = entry.Project
	}

	return m
}

func firstPromptFromSession(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 512*1024), 512*1024)

	for scanner.Scan() {
		var entry sessionJSONLEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.Type == "user" {
			if text, ok := entry.Message.Content.(string); ok {
				if len(text) > 200 {
					return text[:200] + "..."
				}
				return text
			}
		}
	}
	return ""
}
