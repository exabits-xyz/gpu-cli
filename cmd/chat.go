package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/exabits-xyz/gpu-cli/internal/api"
	"github.com/exabits-xyz/gpu-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	chatModel             string
	chatSystem            string
	chatMessagesJSON      string
	chatStream            bool
	chatTemperature       float64
	chatTopP              float64
	chatTopK              int
	chatPresencePenalty   float64
	chatRepetitionPenalty float64
	chatMaxTokens         int
)

var chatCmd = &cobra.Command{
	Use:   "chat [prompt]",
	Short: "Send a chat completion request to an AI model",
	Long: `Sends a chat conversation to POST /chat/completions and prints the result.

The model name must match the model_name attribute from 'egpu model list'.

Provide the conversation either as a positional prompt (a single user message,
optionally preceded by a --system message) or as a full --messages JSON array:

  egpu chat --model MiniMaxAI/MiniMax-M2.7 "hello"
  egpu chat --model MiniMaxAI/MiniMax-M2.7 --system "You are terse." "hello"
  egpu chat --model MiniMaxAI/MiniMax-M2.7 \
            --messages '[{"role":"user","content":"hello"}]'

By default the full completion response is printed as JSON.

With --stream:
  - TTY stdout    — assistant text is printed as it is generated
  - piped stdout  — one JSON chunk object per line (NDJSON) for agent use`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if chatModel == "" {
			exitInvalidArgs(fmt.Errorf("--model is required — find model names with 'egpu model list'"))
			return nil
		}

		messages, err := buildChatMessages(args)
		if err != nil {
			exitInvalidArgs(err)
			return nil
		}

		req := types.ChatCompletionRequest{
			Model:    chatModel,
			Messages: messages,
		}
		if cmd.Flags().Changed("temperature") {
			req.Temperature = &chatTemperature
		}
		if cmd.Flags().Changed("top-p") {
			req.TopP = &chatTopP
		}
		if cmd.Flags().Changed("top-k") {
			req.TopK = &chatTopK
		}
		if cmd.Flags().Changed("presence-penalty") {
			req.PresencePenalty = &chatPresencePenalty
		}
		if cmd.Flags().Changed("repetition-penalty") {
			req.RepetitionPenalty = &chatRepetitionPenalty
		}
		if cmd.Flags().Changed("max-tokens") {
			req.MaxTokens = &chatMaxTokens
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		if chatStream {
			if err := streamChatToStdout(client, req); err != nil {
				exitAPIError(err)
				return nil
			}
			return nil
		}

		completion, err := client.ChatCompletion(req)
		if err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(completion)
		return nil
	},
}

// buildChatMessages assembles the conversation from --messages JSON or from
// the optional --system flag plus the positional prompt.
func buildChatMessages(args []string) ([]types.ChatMessage, error) {
	if chatMessagesJSON != "" {
		if len(args) > 0 || chatSystem != "" {
			return nil, fmt.Errorf("--messages cannot be combined with a positional prompt or --system")
		}
		var messages []types.ChatMessage
		if err := json.Unmarshal([]byte(chatMessagesJSON), &messages); err != nil {
			return nil, fmt.Errorf("--messages must be a JSON array of {role, content} objects: %w", err)
		}
		if len(messages) == 0 {
			return nil, fmt.Errorf("--messages must contain at least one message")
		}
		for i, m := range messages {
			if m.Role == "" || m.Content == "" {
				return nil, fmt.Errorf("--messages[%d] must include non-empty role and content", i)
			}
		}
		return messages, nil
	}

	if len(args) == 0 || args[0] == "" {
		return nil, fmt.Errorf("provide a prompt argument or --messages — e.g. egpu chat --model <name> \"hello\"")
	}

	var messages []types.ChatMessage
	if chatSystem != "" {
		messages = append(messages, types.ChatMessage{Role: "system", Content: chatSystem})
	}
	messages = append(messages, types.ChatMessage{Role: "user", Content: args[0]})
	return messages, nil
}

// streamChatToStdout consumes a streaming completion.
//   - TTY stdout: print assistant content deltas as plain text
//   - piped stdout: emit each raw chunk as one JSON line (NDJSON)
func streamChatToStdout(client *api.Client, req types.ChatCompletionRequest) error {
	tty := stdoutIsTTY()
	enc := json.NewEncoder(os.Stdout)
	printedContent := false

	err := client.ChatCompletionStream(req, func(chunk types.ChatCompletionChunk) error {
		if !tty {
			return enc.Encode(chunk)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				fmt.Print(choice.Delta.Content)
				printedContent = true
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if tty && printedContent {
		fmt.Println()
	}
	return nil
}

// stdoutIsTTY reports whether stdout is an interactive terminal.
func stdoutIsTTY() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func init() {
	chatCmd.Flags().StringVar(&chatModel, "model", "", "Model name from 'egpu model list' (required)")
	chatCmd.Flags().StringVar(&chatSystem, "system", "", "Optional system message prepended to the conversation")
	chatCmd.Flags().StringVar(&chatMessagesJSON, "messages", "", `Full conversation as a JSON array, e.g. '[{"role":"user","content":"hi"}]'`)
	chatCmd.Flags().BoolVar(&chatStream, "stream", false, "Stream the response (TTY: plain text; piped: NDJSON chunks)")
	chatCmd.Flags().Float64Var(&chatTemperature, "temperature", 0, "Sampling temperature, 0–2")
	chatCmd.Flags().Float64Var(&chatTopP, "top-p", 0, "Nucleus sampling probability mass, 0–1")
	chatCmd.Flags().IntVar(&chatTopK, "top-k", 0, "Consider only the top K tokens, -1–200")
	chatCmd.Flags().Float64Var(&chatPresencePenalty, "presence-penalty", 0, "Presence penalty, -2–2")
	chatCmd.Flags().Float64Var(&chatRepetitionPenalty, "repetition-penalty", 0, "Repetition penalty, 0.01–2 (default 1.0)")
	chatCmd.Flags().IntVar(&chatMaxTokens, "max-tokens", 0, "Maximum tokens to generate")

	rootCmd.AddCommand(chatCmd)
}
