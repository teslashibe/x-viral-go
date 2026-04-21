# viral-go

Go implementation of X's algorithm scoring formula. Score draft posts for viral potential, get actionable feedback, and iterate with LLMs to maximize reach.

Based on the weighted scoring formula from [xai-org/x-algorithm](https://github.com/xai-org/x-algorithm) (`home-mixer/scorers/weighted_scorer.rs`).

```
go get github.com/teslashibe/x-viral-go
```

## Quick Start

### Score a Post

```go
import viral "github.com/teslashibe/x-viral-go"

scorer := viral.New()
result := scorer.Score("What's the biggest lesson you've learned building in public?")

fmt.Println(result.Score)           // 72 (0-100)
fmt.Println(result.PositiveFactors) // ["Post asks a question...", ...]
fmt.Println(result.NegativeFactors) // []
fmt.Println(result.Feedback)        // actionable suggestions
```

### Score with Media & Metadata

```go
post := viral.Post{
    Text:      "Here's the architecture of our new system 🧵",
    MediaType: viral.MediaImage,
    IsThread:  true,
}
result := scorer.ScorePost(post)
```

### Score with Author Context

```go
result := scorer.ScoreWithAuthor(post, viral.AuthorProfile{
    IsPremium:      true,
    FollowersCount: 10000,
    FollowingCount: 500,
})
```

### Compare Drafts

```go
results := scorer.ScoreBatch([]viral.Post{
    {Text: "Version A of my post"},
    {Text: "Version B — what do you think?"},
    {Text: "Version C with more detail and a question hook?"},
})
// results[0].Score, results[1].Score, results[2].Score
```

### LLM-Assisted Optimization

```go
scorer := viral.New(viral.WithLLMProvider(myProvider))

result, err := scorer.Optimize(ctx, "The X algorithm is interesting",
    viral.WithMaxIterations(3),
    viral.WithTargetScore(85),
    viral.WithTone("conversational"),
    viral.WithAudience("developers"),
)

fmt.Println(result.Best.Text)       // optimized rewrite
fmt.Println(result.Best.Score.Score) // e.g. 87
fmt.Println(result.Best.Reasoning)  // "Added question hook, data points..."
```

### Generate a Standalone Prompt

Like [willitgoviral.xyz](https://willitgoviral.xyz/)'s "Copy Prompt" feature — paste into ChatGPT, Claude, or Grok:

```go
result := scorer.Score("My draft...")
prompt := scorer.GeneratePrompt(result)
// prompt is a self-contained LLM prompt with score breakdown + suggestions
```

### Custom LLM Provider

Implement the `LLMProvider` interface to plug in any backend:

```go
type LLMProvider interface {
    Complete(ctx context.Context, prompt string) (string, error)
}
```

Or use `LLMProviderFunc` for a quick adapter:

```go
provider := viral.LLMProviderFunc(func(ctx context.Context, prompt string) (string, error) {
    // call OpenAI, Anthropic, Ollama, etc.
    return completion, nil
})
```

## How It Works

X ranks every post with a weighted sum of 19 predicted engagement probabilities:

```
Final Score = Σ (weight_i × P(action_i))
```

The most impactful weights:

| Action | Weight | What drives it |
|---|---|---|
| Reply | 27.0 | Questions, controversy, personal stories |
| Quote | 13.0 | Hot takes, data points, contrarian opinions |
| Retweet | 4.0 | Shareable insights, listicles, data |
| Share | 3.0 | Valuable/reference-worthy content |
| Favorite | 1.0 | Baseline engagement |
| Not Interested | -74.0 | Spam, caps, clickbait |
| Block/Mute | -74.0 | Engagement bait, spam patterns |
| Report | -369.0 | Severe spam/abuse patterns |

Since the actual Grok transformer model is not available outside X, `viral-go` estimates `P(action)` from content signals using rule-based heuristics. External links trigger a 40-70% penalty. Video gets ~10x, images ~2-3x engagement vs text-only.

## Customizing Weights

```go
weights := viral.DefaultWeights()
weights.Reply = 50.0 // increase reply importance
scorer := viral.New(viral.WithWeights(weights))
```

## Zero Dependencies

stdlib only. No OpenAI SDK, no HTTP clients, no external packages. Scoring is pure local computation.
