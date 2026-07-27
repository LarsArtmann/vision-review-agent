package vision

import (
	"charm.land/fantasy"
)

// Conversation holds a multi-turn conversation history for vision analysis.
// It accumulates user and assistant messages so that follow-up questions can
// reference previous context.
//
// Usage:
//
//	conv := vision.NewConversation()
//	conv.AddUserMessage("Describe this UI", img)
//	conv.AddAssistantMessage(result.Text)
//
//	// Follow-up question with conversation context
//	followUp, _ := agent.AnalyzeConversation(ctx, conv, "What about the color contrast?", img)
//	conv.AddUserMessage("What about the color contrast?", img)
//	conv.AddAssistantMessage(followUp.Text)
type Conversation struct {
	messages []fantasy.Message
}

// NewConversation creates a new empty conversation.
func NewConversation() *Conversation {
	return &Conversation{}
}

// AddUserMessage appends a user message with optional images.
// Nil images are filtered out automatically.
func (c *Conversation) AddUserMessage(text string, images ...*ImageSource) *Conversation {
	c.messages = append(c.messages, newMessage(fantasy.MessageRoleUser, text, images...))
	return c
}

// AddAssistantMessage appends an assistant response to the history.
func (c *Conversation) AddAssistantMessage(text string) *Conversation {
	c.messages = append(c.messages, newMessage(fantasy.MessageRoleAssistant, text))
	return c
}

// Messages returns the underlying conversation messages for use with fantasy calls.
func (c *Conversation) Messages() []fantasy.Message {
	return c.messages
}

// Len returns the number of messages in the conversation.
func (c *Conversation) Len() int {
	return len(c.messages)
}

// Clear removes all messages, resetting the conversation to empty.
// Returns the conversation for fluent chaining.
func (c *Conversation) Clear() *Conversation {
	c.messages = nil
	return c
}

// newMessage builds a fantasy.Message with the given role, text, and optional image file parts.
func newMessage(role fantasy.MessageRole, text string, images ...*ImageSource) fantasy.Message {
	valid := filterValidImages(images)
	if len(valid) == 0 {
		return fantasy.Message{
			Role:    role,
			Content: []fantasy.MessagePart{fantasy.TextPart{Text: text}},
		}
	}

	parts := make([]fantasy.MessagePart, 0, len(valid)+1)
	parts = append(parts, fantasy.TextPart{Text: text})

	for _, filePart := range toFileParts(valid) {
		parts = append(parts, filePart)
	}

	return fantasy.Message{
		Role:    role,
		Content: parts,
	}
}
