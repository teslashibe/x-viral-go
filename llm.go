package viral

import (
	"context"
	"fmt"
	"strings"
)

// LLMProvider is the interface for LLM backends used by Optimize and
// OptimizePost. Implementations adapt any LLM (OpenAI, Anthropic, Ollama,
// local, etc.) to viral-go's optimization loop.
//
// Complete should return the raw model completion. The package handles all
// prompt construction and response parsing.
type LLMProvider interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// LLMProviderFunc is an adapter that allows ordinary functions to be used as
// LLMProviders.
type LLMProviderFunc func(ctx context.Context, prompt string) (string, error)

// Complete calls f(ctx, prompt).
func (f LLMProviderFunc) Complete(ctx context.Context, prompt string) (string, error) {
	return f(ctx, prompt)
}

// GeneratePrompt produces a self-contained LLM prompt that includes the post,
// its score breakdown, detected signals, and feedback. The prompt instructs
// the LLM to rewrite the post for higher viral potential and explain its
// changes.
//
// The output can be pasted into ChatGPT, Claude, Grok, or any chat interface,
// matching the "Copy Prompt" feature on willitgoviral.xyz.
func (s *Scorer) GeneratePrompt(result ScoreResult) string {
	return buildOptimizationPrompt(promptInput{
		post:       Post{Text: ""},
		result:     result,
		iteration:  0,
		options:    optimizeOptions{preserveIntent: true},
		standalone: true,
	})
}

// promptInput aggregates everything the prompt builder needs.
type promptInput struct {
	post       Post
	result     ScoreResult
	iteration  int
	options    optimizeOptions
	standalone bool
}

func buildOptimizationPrompt(in promptInput) string {
	var b strings.Builder

	b.WriteString("# X Post Viral Optimizer\n\n")
	b.WriteString("You are an expert at writing X (Twitter) posts that perform well in the X algorithm. ")
	b.WriteString("Your job is to rewrite the post below to maximize its viral score (0-100) according to X's open-source algorithm weights.\n\n")

	b.WriteString("## How X's algorithm scores posts\n\n")
	b.WriteString("X uses a weighted sum of predicted engagement probabilities. The most important weights are:\n\n")
	b.WriteString("- **Reply: 27.0** (highest — questions and conversation drive replies)\n")
	b.WriteString("- **Quote: 13.0** (controversial takes and data points get quoted)\n")
	b.WriteString("- **Retweet: 4.0** (shareable insights get retweeted)\n")
	b.WriteString("- **Share: 3.0**, **DwellTime: 0.5**, **Click: 0.5**, **Favorite: 1.0** (baseline)\n")
	b.WriteString("- **Negative: NotInterested -74, BlockAuthor -74, MuteAuthor -74, Report -369**\n")
	b.WriteString("- **External links** trigger a ~40-70% distribution penalty\n\n")

	b.WriteString("## Current post\n\n")
	if in.post.Text != "" {
		b.WriteString("```\n")
		b.WriteString(in.post.Text)
		b.WriteString("\n```\n\n")
	} else {
		b.WriteString("(post text inserted by user)\n\n")
	}

	b.WriteString(fmt.Sprintf("## Current viral score: %d/100\n\n", in.result.Score))

	if len(in.result.PositiveFactors) > 0 {
		b.WriteString("### What's working\n\n")
		for _, f := range in.result.PositiveFactors {
			b.WriteString("- ")
			b.WriteString(f)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(in.result.NegativeFactors) > 0 {
		b.WriteString("### What's hurting the score\n\n")
		for _, f := range in.result.NegativeFactors {
			b.WriteString("- ")
			b.WriteString(f)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(in.result.Feedback) > 0 {
		b.WriteString("### Specific improvements (in priority order)\n\n")
		for i, sug := range in.result.Feedback {
			if i >= 8 {
				break
			}
			b.WriteString(fmt.Sprintf("%d. [%s impact] %s\n", i+1, sug.Impact, sug.Description))
		}
		b.WriteString("\n")
	}

	if in.options.tone != "" || in.options.audience != "" || len(in.options.constraints) > 0 || in.options.preserveIntent {
		b.WriteString("## Constraints\n\n")
		if in.options.tone != "" {
			b.WriteString("- Tone: ")
			b.WriteString(in.options.tone)
			b.WriteString("\n")
		}
		if in.options.audience != "" {
			b.WriteString("- Target audience: ")
			b.WriteString(in.options.audience)
			b.WriteString("\n")
		}
		if in.options.preserveIntent {
			b.WriteString("- Preserve the core message and intent of the original post.\n")
		}
		for _, c := range in.options.constraints {
			b.WriteString("- ")
			b.WriteString(c)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if in.standalone {
		b.WriteString("## Your task\n\n")
		b.WriteString("Rewrite the post above to maximize its viral score. Apply the improvements listed and any other algorithm-friendly changes. ")
		b.WriteString("Then explain (in 2-3 sentences) what you changed and why.\n\n")
		b.WriteString("Return your response as a single rewritten post followed by your reasoning.\n")
	} else {
		b.WriteString("## Your task\n\n")
		b.WriteString("Rewrite the post to maximize its viral score. Then return your response as a strict JSON object:\n\n")
		b.WriteString("```json\n")
		b.WriteString("{\n")
		b.WriteString("  \"text\": \"<the rewritten post text — no surrounding quotes inside the value>\",\n")
		b.WriteString("  \"reasoning\": \"<2-3 sentences explaining what you changed and why>\"\n")
		b.WriteString("}\n")
		b.WriteString("```\n\n")
		b.WriteString("Return ONLY the JSON. Do not include any other text, markdown, or commentary outside the JSON object.\n")
	}

	return b.String()
}
