package types

// ChatMessage is a single message in a chat conversation.
type ChatMessage struct {
	Role    string `json:"role"` // e.g. "system", "developer", "user", "assistant"
	Content string `json:"content"`
}

// ChatCompletionRequest is the body for POST /chat/completions.
// Optional sampling parameters use pointers so that unset values are omitted
// from the JSON body and the model's defaults apply.
type ChatCompletionRequest struct {
	Model             string        `json:"model"`
	Stream            bool          `json:"stream"`
	Messages          []ChatMessage `json:"messages"`
	Temperature       *float64      `json:"temperature,omitempty"`        // 0–2
	TopP              *float64      `json:"top_p,omitempty"`              // 0–1
	TopK              *int          `json:"top_k,omitempty"`              // -1–200
	PresencePenalty   *float64      `json:"presence_penalty,omitempty"`   // -2–2
	RepetitionPenalty *float64      `json:"repetition_penalty,omitempty"` // 0.01–2
	MaxTokens         *int          `json:"max_tokens,omitempty"`
}

// ChatReasoningDetail is one entry of a message's reasoning_details array.
type ChatReasoningDetail struct {
	ID     string `json:"id"`
	Type   string `json:"type"`   // e.g. "reasoning.text"
	Text   string `json:"text"`
	Format string `json:"format"` // e.g. "openai-responses-v1"
	Index  int    `json:"index"`
}

// ChatResponseMessage is the assistant message inside a completion choice.
type ChatResponseMessage struct {
	Role             string                `json:"role"`
	Content          string                `json:"content"`
	Name             string                `json:"name,omitempty"`
	ReasoningContent string                `json:"reasoning_content,omitempty"`
	ReasoningDetails []ChatReasoningDetail `json:"reasoning_details,omitempty"`
}

// ChatChoice is one completion alternative in a non-streaming response.
type ChatChoice struct {
	Index        int                 `json:"index"`
	Message      ChatResponseMessage `json:"message"`
	FinishReason string              `json:"finish_reason"`
}

// ChatTokenDetails breaks down completion token usage by type.
type ChatTokenDetails struct {
	TextTokens               int `json:"text_tokens"`
	ReasoningTokens          int `json:"reasoning_tokens"`
	AudioTokens              int `json:"audio_tokens"`
	ImageTokens              int `json:"image_tokens"`
	VideoTokens              int `json:"video_tokens"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
}

// ChatUsage reports token consumption for a chat completion.
type ChatUsage struct {
	PromptTokens            int               `json:"prompt_tokens"`
	CompletionTokens        int               `json:"completion_tokens"`
	TotalTokens             int               `json:"total_tokens"`
	CompletionTokensDetails *ChatTokenDetails `json:"completion_tokens_details,omitempty"`
	PromptTokensDetails     *ChatTokenDetails `json:"prompt_tokens_details,omitempty"`
}

// ChatCompletionResponse is the non-streaming response from POST /chat/completions.
// Unlike other Exabits endpoints, this response is NOT wrapped in the
// {status, message, data} envelope — it follows the OpenAI chat format.
type ChatCompletionResponse struct {
	ID                string       `json:"id"`
	Object            string       `json:"object"` // "chat.completion"
	Created           int64        `json:"created"`
	Model             string       `json:"model"`
	Choices           []ChatChoice `json:"choices"`
	SystemFingerprint string       `json:"system_fingerprint,omitempty"`
	Usage             *ChatUsage   `json:"usage,omitempty"`
}

// ChatDelta is the incremental payload inside a streaming chunk choice.
type ChatDelta struct {
	Role             string                `json:"role,omitempty"`
	Content          string                `json:"content,omitempty"`
	ReasoningContent string                `json:"reasoning_content,omitempty"`
	ReasoningDetails []ChatReasoningDetail `json:"reasoning_details,omitempty"`
}

// ChatChunkChoice is one choice inside a streaming chunk.
type ChatChunkChoice struct {
	Index        int       `json:"index"`
	Delta        ChatDelta `json:"delta"`
	FinishReason *string   `json:"finish_reason"`
}

// ChatCompletionChunk is a single SSE "data:" event in a streaming response.
// The final chunk before [DONE] carries Usage and an empty Choices array.
type ChatCompletionChunk struct {
	ID                string            `json:"id"`
	Object            string            `json:"object"` // "chat.completion.chunk"
	Created           int64             `json:"created"`
	Model             string            `json:"model"`
	Choices           []ChatChunkChoice `json:"choices"`
	SystemFingerprint string            `json:"system_fingerprint,omitempty"`
	Usage             *ChatUsage        `json:"usage,omitempty"`
}
