package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AddExamplePrompts adds example prompts to the MCP server
func AddExamplePrompts(s *server.MCPServer) {
	// LLM Backend chat prompt
	lmChatPrompt := mcp.NewPrompt("lm_chat_prompt",
		mcp.WithPromptDescription("Prompt template for generating responses using LLM Backend"),
		mcp.WithArgument("topic",
			mcp.ArgumentDescription("Conversation topic"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("style",
			mcp.ArgumentDescription("Response style (e.g.: professional, friendly, humorous)"),
		),
	)

	s.AddPrompt(lmChatPrompt, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := request.Params.Arguments
		topic := args["topic"]
		if topic == "" {
			topic = "general conversation"
		}

		style := args["style"]
		if style == "" {
			style = "friendly"
		}

		promptText := fmt.Sprintf("Please discuss the topic of '%s' with me in a %s style. Please respond in English.", topic, style)

		return &mcp.GetPromptResult{
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleUser,
					Content: mcp.TextContent{
						Type: "text",
						Text: promptText,
					},
				},
			},
		}, nil
	})

	log.Println("Registered prompt: lm_chat_prompt")
}