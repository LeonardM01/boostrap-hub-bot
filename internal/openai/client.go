package openai

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// Client wraps the OpenAI API client for point calculations
type Client struct {
	client *openai.Client
}

// New creates a new OpenAI client
func New(apiKey string) *Client {
	if apiKey == "" {
		log.Println("Warning: OpenAI API key not configured, point calculations will be disabled")
		return nil
	}

	return &Client{
		client: openai.NewClient(apiKey),
	}
}

// CalculatePoints uses OpenAI to estimate task difficulty and assign points (1-10 scale)
func (c *Client) CalculatePoints(taskTitle, taskDescription string) (int, error) {
	if c == nil {
		// If client is nil (no API key), return default points
		log.Println("OpenAI client not configured, using default points")
		return 5, nil
	}

	ctx := context.Background()

	// Build the prompt
	prompt := fmt.Sprintf(`You are a task difficulty estimator for solo founders and entrepreneurs.
Analyze the following task and assign a difficulty score from 1-10 based on:
- Time investment required (1=<1hr, 5=1-2 days, 10=1+ weeks)
- Complexity and skill required
- Impact on business outcomes

Task Title: %s
Task Description: %s

Respond with ONLY a single number between 1 and 10, nothing else.`, taskTitle, taskDescription)

	req := openai.ChatCompletionRequest{
		Model: openai.GPT4oMini, // Using GPT-4o-mini for cost efficiency
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Temperature: 0.3, // Lower temperature for more consistent results
		MaxTokens:   10,  // We only need a single number
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return 0, fmt.Errorf("no response from OpenAI")
	}

	// Extract the number from the response
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	points, err := extractPoints(content)
	if err != nil {
		log.Printf("Failed to parse OpenAI response '%s': %v", content, err)
		// Fallback to default
		return 5, nil
	}

	// Validate range
	if points < 1 {
		points = 1
	} else if points > 10 {
		points = 10
	}

	log.Printf("OpenAI calculated %d points for task: %s", points, taskTitle)
	return points, nil
}

// extractPoints extracts a number from OpenAI's response
func extractPoints(content string) (int, error) {
	// Try to extract any number from the response
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(content)

	if match == "" {
		return 0, fmt.Errorf("no number found in response")
	}

	points, err := strconv.Atoi(match)
	if err != nil {
		return 0, fmt.Errorf("failed to convert to integer: %w", err)
	}

	return points, nil
}
