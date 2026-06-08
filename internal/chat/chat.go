package chat

import "time"

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type Conversation struct {
	Agent    string    `json:"agent"`
	System   string    `json:"system"`
	Messages []Message `json:"messages"`
}

func NewConversation(agent, systemPrompt string) *Conversation {
	return &Conversation{
		Agent:    agent,
		System:   systemPrompt,
		Messages: []Message{},
	}
}

func (c *Conversation) Add(role Role, content string) {
	c.Messages = append(c.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

// APIMessages returns the messages formatted for the API (role + content pairs)
func (c *Conversation) APIMessages() []map[string]string {
	msgs := make([]map[string]string, len(c.Messages))
	for i, m := range c.Messages {
		msgs[i] = map[string]string{"role": string(m.Role), "content": m.Content}
	}
	return msgs
}
