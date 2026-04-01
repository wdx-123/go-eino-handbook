package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/google/uuid"
)

type Session struct {
	ID        string
	CreatedAt time.Time
	filePath  string
	messages  []*schema.Message
}

func (s *Session) Append(msg *schema.Message) error {
	s.messages = append(s.messages, msg)

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

func (s *Session) GetMessages() []*schema.Message {
	result := make([]*schema.Message, len(s.messages))
	copy(result, s.messages)
	return result
}

func (s *Session) Title() string {
	for _, msg := range s.messages {
		if msg.Role == schema.User && msg.Content != "" {
			title := msg.Content
			if len([]rune(title)) > 40 {
				title = string([]rune(title)[:40]) + "..."
			}
			return title
		}
	}
	return "New Session"
}

type Store struct {
	dir   string
	cache map[string]*Session
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{
		dir:   dir,
		cache: make(map[string]*Session),
	}, nil
}

func (s *Store) GetOrCreate(id string) (*Session, error) {
	if sess, ok := s.cache[id]; ok {
		return sess, nil
	}

	filePath := filepath.Join(s.dir, id+".jsonl")
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			sess, createErr := createSession(id, filePath)
			if createErr != nil {
				return nil, createErr
			}
			s.cache[id] = sess
			return sess, nil
		}
		return nil, err
	}

	sess, err := loadSession(filePath)
	if err != nil {
		return nil, err
	}

	s.cache[id] = sess
	return sess, nil
}

type sessionHeader struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func createSession(id, filePath string) (*Session, error) {
	header := sessionHeader{
		Type:      "session",
		ID:        id,
		CreatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(filePath, append(data, '\n'), 0o644); err != nil {
		return nil, err
	}

	return &Session{
		ID:        id,
		CreatedAt: header.CreatedAt,
		filePath:  filePath,
		messages:  make([]*schema.Message, 0),
	}, nil
}

func loadSession(filePath string) (*Session, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty session file: %s", filePath)
	}

	var header sessionHeader
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return nil, err
	}

	sess := &Session{
		ID:        header.ID,
		CreatedAt: header.CreatedAt,
		filePath:  filePath,
		messages:  make([]*schema.Message, 0),
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg schema.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		sess.messages = append(sess.messages, &msg)
	}

	return sess, scanner.Err()
}

func main() {
	var sessionID string
	flag.StringVar(&sessionID, "session", "", "session ID")
	flag.Parse()

	ctx := context.Background()

	cm, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  mustEnv("DASHSCOPE_API_KEY"),
		Model:   envOrDefault("QWEN_MODEL", "qwen3.5-flash"),
	})
	if err != nil {
		log.Fatalf("new qwen chat model failed: %v", err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MemoryDemoAgent",
		Description: "ChatModelAgent with persistent session.",
		Instruction: "你是一个简洁、专业的 Eino 学习助手。",
		Model:       cm,
	})
	if err != nil {
		log.Fatalf("new chat model agent failed: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	store, err := NewStore(envOrDefault("SESSION_DIR", "./data/sessions"))
	if err != nil {
		log.Fatalf("new store failed: %v", err)
	}

	if sessionID == "" {
		sessionID = uuid.NewString()
		fmt.Printf("Created new session: %s\n", sessionID)
	} else {
		fmt.Printf("Resuming session: %s\n", sessionID)
	}

	session, err := store.GetOrCreate(sessionID)
	if err != nil {
		log.Fatalf("get or create session failed: %v", err)
	}

	fmt.Printf("Session title: %s\n", session.Title())
	fmt.Println("Enter your message (empty line to exit):")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("you> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		userMsg := schema.UserMessage(line)
		if err := session.Append(userMsg); err != nil {
			log.Fatalf("append user message failed: %v", err)
		}

		events := runner.Run(ctx, session.GetMessages())
		content, err := printAndCollectAssistant(events)
		if err != nil {
			log.Fatalf("run agent failed: %v", err)
		}

		assistantMsg := schema.AssistantMessage(content, nil)
		if err := session.Append(assistantMsg); err != nil {
			log.Fatalf("append assistant message failed: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nSession saved: %s\n", sessionID)
	fmt.Printf("Resume with: go run ./memory-session --session %s\n", sessionID)
}

func printAndCollectAssistant(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		if mv.Role != schema.Assistant && mv.Role != "" {
			continue
		}

		if mv.IsStreaming && mv.MessageStream != nil {
			mv.MessageStream.SetAutomaticClose()
			for {
				chunk, err := mv.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return "", err
				}
				if chunk != nil && chunk.Content != "" {
					fmt.Print(chunk.Content)
					sb.WriteString(chunk.Content)
				}
			}
			fmt.Println()
			continue
		}

		if mv.Message != nil {
			fmt.Println(mv.Message.Content)
			sb.WriteString(mv.Message.Content)
		}
	}

	return sb.String(), nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is empty", key)
	}
	return v
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
