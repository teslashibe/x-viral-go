package viral

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// optimizeOptions holds the tunables for an optimization run.
type optimizeOptions struct {
	maxIterations  int
	targetScore    int
	tone           string
	audience       string
	constraints    []string
	preserveIntent bool
}

// OptimizeOption configures an Optimize or OptimizePost call.
type OptimizeOption func(*optimizeOptions)

// WithMaxIterations sets the maximum number of LLM rewrite attempts.
// Default: 3.
func WithMaxIterations(n int) OptimizeOption {
	return func(o *optimizeOptions) {
		if n > 0 {
			o.maxIterations = n
		}
	}
}

// WithTargetScore sets a score threshold (0-100) at which optimization stops
// early. Default: 80.
func WithTargetScore(score int) OptimizeOption {
	return func(o *optimizeOptions) {
		if score > 0 && score <= 100 {
			o.targetScore = score
		}
	}
}

// WithTone constrains the LLM rewrites to a specific voice
// (e.g. "professional", "conversational", "provocative").
func WithTone(tone string) OptimizeOption {
	return func(o *optimizeOptions) { o.tone = tone }
}

// WithAudience instructs the LLM to tailor rewrites to a target audience
// (e.g. "Go developers", "indie founders").
func WithAudience(audience string) OptimizeOption {
	return func(o *optimizeOptions) { o.audience = audience }
}

// WithConstraints adds hard constraints the LLM must respect
// (e.g. "keep under 280 characters", "must include @somehandle").
func WithConstraints(constraints ...string) OptimizeOption {
	return func(o *optimizeOptions) { o.constraints = append(o.constraints, constraints...) }
}

// WithPreserveIntent toggles whether the LLM must keep the original message's
// core intent. Default: true.
func WithPreserveIntent(preserve bool) OptimizeOption {
	return func(o *optimizeOptions) { o.preserveIntent = preserve }
}

func defaultOptimizeOptions() optimizeOptions {
	return optimizeOptions{
		maxIterations:  3,
		targetScore:    80,
		preserveIntent: true,
	}
}

// Optimize iteratively rewrites a plain-text post using an LLM to maximize
// its viral score. Returns the original score, every iteration, and the
// best-scoring version.
//
// Requires an LLMProvider configured via WithLLMProvider. Each iteration's
// prompt includes the previous score breakdown, detected signals, and
// feedback so the LLM can target specific improvements.
func (s *Scorer) Optimize(ctx context.Context, text string, opts ...OptimizeOption) (*OptimizeResult, error) {
	return s.OptimizePost(ctx, Post{Text: text}, opts...)
}

// OptimizePost is like Optimize but accepts a structured Post with media
// type, format flags, and other metadata so the LLM is aware of full context.
func (s *Scorer) OptimizePost(ctx context.Context, post Post, opts ...OptimizeOption) (*OptimizeResult, error) {
	if s.provider == nil {
		return nil, ErrNoLLMProvider
	}
	if strings.TrimSpace(post.Text) == "" {
		return nil, ErrEmptyText
	}

	options := defaultOptimizeOptions()
	for _, o := range opts {
		o(&options)
	}

	enrichPost(&post)
	original := scorePost(post, nil, s.weights, s.normalizer)

	result := &OptimizeResult{Original: original}
	best := Iteration{Number: 0, Text: post.Text, Score: original, Reasoning: "original"}

	current := post
	currentScore := original

	for i := 1; i <= options.maxIterations; i++ {
		if err := ctx.Err(); err != nil {
			result.Best = best
			return result, fmt.Errorf("%w: %v", ErrContextCanceled, err)
		}

		if currentScore.Score >= options.targetScore && i > 1 {
			break
		}

		prompt := buildOptimizationPrompt(promptInput{
			post:      current,
			result:    currentScore,
			iteration: i,
			options:   options,
		})

		completion, err := s.provider.Complete(ctx, prompt)
		if err != nil {
			result.Best = best
			return result, fmt.Errorf("%w: iteration %d: %v", ErrLLMFailed, i, err)
		}

		rewritten, reasoning, parseErr := parseLLMResponse(completion)
		if parseErr != nil {
			result.Best = best
			return result, fmt.Errorf("%w: iteration %d: %v", ErrLLMParseFailed, i, parseErr)
		}

		newPost := current
		newPost.Text = rewritten
		enrichPost(&newPost)
		newScore := scorePost(newPost, nil, s.weights, s.normalizer)

		iter := Iteration{
			Number:    i,
			Text:      rewritten,
			Score:     newScore,
			Reasoning: reasoning,
		}
		result.Iterations = append(result.Iterations, iter)

		if newScore.Score > best.Score.Score {
			best = iter
		}

		current = newPost
		currentScore = newScore

		if newScore.Score >= options.targetScore {
			break
		}
	}

	result.Best = best

	if best.Score.Score < options.targetScore && len(result.Iterations) >= options.maxIterations {
		return result, ErrMaxIterations
	}

	return result, nil
}

// parseLLMResponse extracts the rewritten text and reasoning from an LLM
// completion. It tries strict JSON first, then JSON inside fenced code blocks,
// then the first {...} block anywhere in the response.
func parseLLMResponse(completion string) (text, reasoning string, err error) {
	completion = strings.TrimSpace(completion)
	if completion == "" {
		return "", "", errors.New("empty completion")
	}

	jsonStr := extractJSON(completion)
	if jsonStr == "" {
		return "", "", fmt.Errorf("no JSON object found in completion")
	}

	var resp struct {
		Text      string `json:"text"`
		Reasoning string `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return "", "", fmt.Errorf("json unmarshal: %w", err)
	}

	if strings.TrimSpace(resp.Text) == "" {
		return "", "", errors.New("response missing 'text' field")
	}

	return resp.Text, resp.Reasoning, nil
}

// extractJSON returns the first balanced {...} block in s, stripping fenced
// markdown code blocks if present.
func extractJSON(s string) string {
	if idx := strings.Index(s, "```"); idx >= 0 {
		rest := s[idx+3:]
		if strings.HasPrefix(rest, "json") {
			rest = rest[4:]
		}
		if end := strings.Index(rest, "```"); end >= 0 {
			s = rest[:end]
		}
	}

	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	inStr := false
	escape := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if escape {
			escape = false
			continue
		}
		if inStr {
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inStr = false
			}
			continue
		}
		switch ch {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}
