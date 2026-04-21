// Package viral provides a Go implementation of X's algorithm scoring formula.
//
// It scores draft posts for viral potential and provides structured feedback,
// with optional LLM-in-the-loop optimization to iteratively rewrite posts for
// maximum algorithmic reach.
//
// The scoring formula mirrors the weighted scorer from xai-org/x-algorithm
// (home-mixer/scorers/weighted_scorer.rs):
//
//	final_score = Σ (weight_i × P(action_i)) + offset
//
// Since the actual Grok transformer that predicts P(action_i) is not available
// outside X's infrastructure, viral-go estimates these probabilities from
// content signals (question hooks, format, length, media, links, etc.) using
// rule-based heuristics derived from public algorithm research.
//
// No API keys, no auth, no network calls. Scoring is pure local computation.
//
// Quick start:
//
//	scorer := viral.New()
//	result := scorer.Score("My draft post text...")
//	fmt.Println(result.Score)        // 0-100 viral score
//	fmt.Println(result.Feedback)     // actionable suggestions
//
// Structured scoring with metadata:
//
//	post := viral.Post{
//	    Text:      "Check out this thread 🧵",
//	    MediaType: viral.MediaImage,
//	    IsThread:  true,
//	}
//	result := scorer.ScorePost(post)
//
// LLM-assisted optimization:
//
//	scorer := viral.New(viral.WithLLMProvider(myProvider))
//	opt, _ := scorer.Optimize(ctx, "draft text",
//	    viral.WithMaxIterations(3),
//	    viral.WithTargetScore(85),
//	    viral.WithTone("conversational"),
//	)
//	fmt.Println(opt.Best.Text)       // best rewrite
//	fmt.Println(opt.Best.Score.Score)
//
// Generate a standalone LLM prompt (like willitgoviral.xyz's "Copy Prompt"):
//
//	prompt := scorer.GeneratePrompt(result)
//	// paste prompt into ChatGPT, Claude, Grok, etc.
package viral
