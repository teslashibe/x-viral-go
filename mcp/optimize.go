package mcp

import (
	"context"
	"strings"

	"github.com/teslashibe/mcptool"
	viral "github.com/teslashibe/x-viral-go"
)

// OptimizeOptionsInput captures the optional optimization tunables shared by
// xviral_optimize_text and xviral_optimize_post.
type OptimizeOptionsInput struct {
	MaxIterations  int      `json:"max_iterations,omitempty" jsonschema:"description=max LLM rewrite attempts,minimum=1,maximum=10,default=3"`
	TargetScore    int      `json:"target_score,omitempty" jsonschema:"description=score threshold (0-100) at which optimization stops early,minimum=1,maximum=100,default=80"`
	Tone           string   `json:"tone,omitempty" jsonschema:"description=voice constraint for rewrites (e.g. professional, conversational, provocative)"`
	Audience       string   `json:"audience,omitempty" jsonschema:"description=target audience for rewrites (e.g. Go developers, indie founders)"`
	Constraints    []string `json:"constraints,omitempty" jsonschema:"description=hard constraints the LLM must respect (e.g. 'keep under 280 chars')"`
	PreserveIntent *bool    `json:"preserve_intent,omitempty" jsonschema:"description=keep the original message's core intent (default true)"`
}

// OptimizeTextInput is the typed input for xviral_optimize_text.
type OptimizeTextInput struct {
	Text    string               `json:"text" jsonschema:"description=draft post text to iteratively rewrite via the configured LLM provider,required"`
	Options OptimizeOptionsInput `json:"options,omitempty" jsonschema:"description=optional optimization tunables (max iterations, target score, tone, etc.)"`
}

// OptimizePostInput is the typed input for xviral_optimize_post.
type OptimizePostInput struct {
	Post    PostInput            `json:"post" jsonschema:"description=structured draft post (text + media + format flags) to iteratively rewrite,required"`
	Options OptimizeOptionsInput `json:"options,omitempty" jsonschema:"description=optional optimization tunables (max iterations, target score, tone, etc.)"`
}

// GeneratePromptInput is the typed input for xviral_generate_prompt.
type GeneratePromptInput struct {
	Text string `json:"text" jsonschema:"description=draft post text to score and embed in a self-contained LLM optimization prompt,required"`
}

func optimizeText(ctx context.Context, s *viral.Scorer, in OptimizeTextInput) (any, error) {
	if strings.TrimSpace(in.Text) == "" {
		return nil, &mcptool.Error{Code: "invalid_input", Message: "text must not be empty"}
	}
	res, err := s.Optimize(ctx, in.Text, optimizeOpts(in.Options)...)
	return optimizeResultOrError(res, err)
}

func optimizeStructuredPost(ctx context.Context, s *viral.Scorer, in OptimizePostInput) (any, error) {
	post, err := toViralPost(in.Post)
	if err != nil {
		return nil, err
	}
	res, err := s.OptimizePost(ctx, post, optimizeOpts(in.Options)...)
	return optimizeResultOrError(res, err)
}

func generatePrompt(_ context.Context, s *viral.Scorer, in GeneratePromptInput) (any, error) {
	if strings.TrimSpace(in.Text) == "" {
		return nil, &mcptool.Error{Code: "invalid_input", Message: "text must not be empty"}
	}
	scored := s.Score(in.Text)
	prompt := s.GeneratePrompt(scored)
	return map[string]any{
		"prompt": prompt,
		"score":  scored,
	}, nil
}

// optimizeResultOrError surfaces the partial OptimizeResult even on
// recoverable errors (ErrMaxIterations) so the agent can still see the best
// iteration found before giving up.
func optimizeResultOrError(res *viral.OptimizeResult, err error) (any, error) {
	switch {
	case err == nil:
		return res, nil
	case res != nil:
		return map[string]any{
			"result": res,
			"error":  err.Error(),
		}, nil
	default:
		return nil, err
	}
}

func optimizeOpts(in OptimizeOptionsInput) []viral.OptimizeOption {
	var opts []viral.OptimizeOption
	if in.MaxIterations > 0 {
		opts = append(opts, viral.WithMaxIterations(in.MaxIterations))
	}
	if in.TargetScore > 0 {
		opts = append(opts, viral.WithTargetScore(in.TargetScore))
	}
	if in.Tone != "" {
		opts = append(opts, viral.WithTone(in.Tone))
	}
	if in.Audience != "" {
		opts = append(opts, viral.WithAudience(in.Audience))
	}
	if len(in.Constraints) > 0 {
		opts = append(opts, viral.WithConstraints(in.Constraints...))
	}
	if in.PreserveIntent != nil {
		opts = append(opts, viral.WithPreserveIntent(*in.PreserveIntent))
	}
	return opts
}

var optimizeTools = []mcptool.Tool{
	mcptool.Define[*viral.Scorer, OptimizeTextInput](
		"xviral_optimize_text",
		"Iteratively rewrite a plain-text post via the host's LLM provider to maximize its viral score",
		"Optimize",
		optimizeText,
	),
	mcptool.Define[*viral.Scorer, OptimizePostInput](
		"xviral_optimize_post",
		"Iteratively rewrite a structured post (text + media + flags) via the host's LLM provider for max viral score",
		"OptimizePost",
		optimizeStructuredPost,
	),
	mcptool.Define[*viral.Scorer, GeneratePromptInput](
		"xviral_generate_prompt",
		"Score text and emit a self-contained LLM prompt the agent can use to rewrite the post for higher reach",
		"GeneratePrompt",
		generatePrompt,
	),
}
