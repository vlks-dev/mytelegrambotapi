package models

type ErrorR1Message struct {
	Message  string `json:"message"`
	Code     int    `json:"code"`
	Metadata struct {
		Headers      RateLimitHeaders `json:"headers"`
		ProviderName interface{}      `json:"provider_name"`
	} `json:"metadata"`
}

type RateLimitHeaders struct {
	XRateLimitLimit     string `json:"X-RateLimit-Limit"`
	XRateLimitRemaining string `json:"X-RateLimit-Remaining"`
	XRateLimitReset     string `json:"X-RateLimit-Reset"`
}

type CompletionResponse struct {
	ID       string   `json:"id"`
	Provider string   `json:"provider"`
	Model    string   `json:"model"`
	Object   string   `json:"object"`
	Created  int      `json:"created"`
	Choices  []Choice `json:"choices"`
	Usage    Usage    `json:"usage"`
}

type Choice struct {
	Logprobs           interface{} `json:"logprobs"`
	FinishReason       string      `json:"finish_reason"`
	NativeFinishReason string      `json:"native_finish_reason"`
	Index              int         `json:"index"`
	Message            R1Message   `json:"message"`
}

type R1Message struct {
	Role      string      `json:"role"`
	Content   string      `json:"content"`
	Refusal   interface{} `json:"refusal"`
	Reasoning string      `json:"reasoning"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
