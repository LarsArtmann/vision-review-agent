// Package visionutil provides internal helpers for the vision package.
package visionutil

import (
	"encoding/json"
	"fmt"

	"charm.land/fantasy"
)

// AppendSystemAndPrompt builds a prompt with optional system message and files.
func AppendSystemAndPrompt(
	systemPrompt, userPrompt string,
	files []fantasy.FilePart,
) fantasy.Prompt {
	var prompt fantasy.Prompt
	if systemPrompt != "" {
		prompt = append(prompt, fantasy.NewSystemMessage(systemPrompt))
	}
	prompt = append(prompt, fantasy.NewUserMessage(userPrompt, files...))
	return prompt
}

// UnmarshalToType converts an object to a specific type using JSON round-tripping.
func UnmarshalToType(obj, target any) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("marshal object: %w", err)
	}
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("unmarshal into target: %w", err)
	}
	return nil
}
