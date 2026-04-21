package viral

import "errors"

// Sentinel errors.
var (
	ErrEmptyText       = errors.New("viral: post text is empty")
	ErrNoLLMProvider   = errors.New("viral: LLMProvider required for Optimize — use WithLLMProvider()")
	ErrLLMFailed       = errors.New("viral: LLM completion failed")
	ErrLLMParseFailed  = errors.New("viral: failed to parse LLM response as JSON")
	ErrMaxIterations   = errors.New("viral: reached max iterations without hitting target score")
	ErrContextCanceled = errors.New("viral: context canceled during optimization")
)
