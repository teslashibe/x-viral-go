package viral

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestScoreEmptyText(t *testing.T) {
	s := New()
	result := s.Score("")
	if result.Score != 0 {
		t.Errorf("empty text: want score 0, got %d", result.Score)
	}
}

func TestScoreMinimalText(t *testing.T) {
	s := New()
	result := s.Score("GM")
	if result.Score > 30 {
		t.Errorf("'GM' should score low, got %d", result.Score)
	}
	if len(result.NegativeFactors) == 0 {
		t.Error("'GM' should have negative factors")
	}
}

func TestScoreHighQualityPost(t *testing.T) {
	s := New()
	post := `I spent 6 months building a Go package that scores X posts using the actual algorithm weights.

Here's what I learned about what makes content go viral:

1. Replies are weighted 27x more than likes
2. External links kill your reach by 40-70%
3. Questions at the end are the single highest-impact change

What's the most counterintuitive thing you've learned about social media algorithms?`

	result := s.Score(post)
	if result.Score < 60 {
		t.Errorf("high-quality post should score ≥60, got %d (raw=%.3f)", result.Score, result.RawScore)
	}
	if len(result.PositiveFactors) == 0 {
		t.Error("high-quality post should have positive factors")
	}
}

func TestScoreExternalLink(t *testing.T) {
	s := New()
	result := s.Score("Check out my blog https://myblog.com/post")
	hasSignal := false
	for _, sig := range result.Signals {
		if sig.Name == "external_link" {
			hasSignal = true
			break
		}
	}
	if !hasSignal {
		t.Error("should detect external_link signal")
	}
}

func TestScoreHashtagStuffing(t *testing.T) {
	s := New()
	result := s.Score("Great day! #go #golang #dev #code #tech #programming")
	hasSignal := false
	for _, sig := range result.Signals {
		if sig.Name == "hashtag_stuffing" {
			hasSignal = true
			break
		}
	}
	if !hasSignal {
		t.Error("should detect hashtag_stuffing signal")
	}
}

func TestScoreQuestionHook(t *testing.T) {
	s := New()
	result := s.Score("What's your biggest challenge with Go concurrency?")
	hasSignal := false
	for _, sig := range result.Signals {
		if sig.Name == "question_hook" {
			hasSignal = true
			break
		}
	}
	if !hasSignal {
		t.Error("should detect question_hook signal")
	}
}

func TestScoreSpamPatterns(t *testing.T) {
	s := New()
	result := s.Score("Follow for follow! Like and retweet for a chance to win!")
	hasSignal := false
	for _, sig := range result.Signals {
		if sig.Name == "spam_patterns" {
			hasSignal = true
			break
		}
	}
	if !hasSignal {
		t.Error("should detect spam_patterns signal")
	}
	if result.Score > 30 {
		t.Errorf("spam post should score low, got %d", result.Score)
	}
}

func TestScorePostWithMedia(t *testing.T) {
	s := New()
	text := "Here's the architecture diagram for our new system"

	textOnly := s.ScorePost(Post{Text: text})
	withImage := s.ScorePost(Post{Text: text, MediaType: MediaImage})
	withVideo := s.ScorePost(Post{Text: text, MediaType: MediaVideo})

	if withImage.Score <= textOnly.Score {
		t.Errorf("image post (%d) should score higher than text-only (%d)", withImage.Score, textOnly.Score)
	}
	if withVideo.Score <= withImage.Score {
		t.Errorf("video post (%d) should score higher than image (%d)", withVideo.Score, withImage.Score)
	}
}

func TestScoreWithAuthor(t *testing.T) {
	s := New()
	post := Post{Text: "Building in public: shipped v0.1 today!"}

	noAuthor := s.ScorePost(post)
	premiumAuthor := s.ScoreWithAuthor(post, AuthorProfile{
		IsPremium:      true,
		FollowersCount: 10000,
		FollowingCount: 500,
	})

	if premiumAuthor.Score <= noAuthor.Score {
		t.Errorf("premium author (%d) should score higher than no author (%d)", premiumAuthor.Score, noAuthor.Score)
	}

	hasAuthorSignal := false
	for _, sig := range premiumAuthor.Signals {
		if sig.Category == CategoryAuthor {
			hasAuthorSignal = true
			break
		}
	}
	if !hasAuthorSignal {
		t.Error("should have author category signal")
	}
}

func TestScoreBatch(t *testing.T) {
	s := New()
	posts := []Post{
		{Text: "GM"},
		{Text: "What's your take on the new Go generics patterns? I've been experimenting and found some surprising results."},
		{Text: "Follow for follow! #f4f"},
	}

	results := s.ScoreBatch(posts)

	if len(results) != 3 {
		t.Fatalf("want 3 results, got %d", len(results))
	}
	if results[1].Score <= results[0].Score {
		t.Errorf("quality post (%d) should score higher than 'GM' (%d)", results[1].Score, results[0].Score)
	}
	if results[2].Score >= results[1].Score {
		t.Errorf("spam post (%d) should score lower than quality post (%d)", results[2].Score, results[1].Score)
	}
}

func TestScoreContainsActionScores(t *testing.T) {
	s := New()
	result := s.Score("Building a new open-source project. What features would you want?")
	if result.ActionScores.Reply <= 0 {
		t.Error("reply probability should be > 0")
	}
	if result.ActionScores.Favorite <= 0 {
		t.Error("favorite probability should be > 0")
	}
}

func TestScoreFeedbackPresent(t *testing.T) {
	s := New()
	result := s.Score("ok")
	if len(result.Feedback) == 0 {
		t.Error("low-scoring post should have feedback suggestions")
	}
}

func TestCustomWeights(t *testing.T) {
	w := DefaultWeights()
	w.Reply = 100.0

	s := New(WithWeights(w))
	result := s.Score("What do you think about this? I'd love to hear your perspective!")

	sDefault := New()
	resultDefault := sDefault.Score("What do you think about this? I'd love to hear your perspective!")

	if result.RawScore <= resultDefault.RawScore {
		t.Errorf("custom reply weight (100) should produce higher raw score: custom=%.3f default=%.3f",
			result.RawScore, resultDefault.RawScore)
	}
}

func TestCustomNormalizer(t *testing.T) {
	always50 := func(raw float64) int { return 50 }
	s := New(WithScoreNormalizer(always50))
	result := s.Score("Anything at all")
	if result.Score != 50 {
		t.Errorf("custom normalizer should return 50, got %d", result.Score)
	}
}

func TestWeightsSum(t *testing.T) {
	w := DefaultWeights()
	sum := w.WeightsSum()
	if sum <= 0 {
		t.Errorf("positive weights sum should be > 0, got %f", sum)
	}
	negSum := w.NegativeWeightsSum()
	if negSum <= 0 {
		t.Errorf("negative weights sum should be > 0, got %f", negSum)
	}
}

func TestMediaString(t *testing.T) {
	tests := []struct {
		m    Media
		want string
	}{
		{MediaNone, "none"},
		{MediaImage, "image"},
		{MediaVideo, "video"},
		{MediaPoll, "poll"},
		{MediaGIF, "gif"},
	}
	for _, tt := range tests {
		if got := tt.m.String(); got != tt.want {
			t.Errorf("Media(%d).String() = %q, want %q", tt.m, got, tt.want)
		}
	}
}

func TestGeneratePrompt(t *testing.T) {
	s := New()
	result := s.Score("My draft post about Go performance")
	prompt := s.GeneratePrompt(result)

	if !strings.Contains(prompt, "Reply: 27.0") {
		t.Error("prompt should contain algorithm weight info")
	}
	if !strings.Contains(prompt, "Viral Optimizer") {
		t.Error("prompt should contain title")
	}
	if len(prompt) < 200 {
		t.Errorf("prompt should be substantial, got %d chars", len(prompt))
	}
}

// === Optimize tests (mock LLM) ===

type mockLLM struct {
	responses []string
	calls     int
}

func (m *mockLLM) Complete(_ context.Context, prompt string) (string, error) {
	if m.calls >= len(m.responses) {
		return "", fmt.Errorf("no more mock responses")
	}
	resp := m.responses[m.calls]
	m.calls++
	return resp, nil
}

func TestOptimizeNoProvider(t *testing.T) {
	s := New()
	_, err := s.Optimize(context.Background(), "hello")
	if err != ErrNoLLMProvider {
		t.Errorf("want ErrNoLLMProvider, got %v", err)
	}
}

func TestOptimizeEmptyText(t *testing.T) {
	s := New(WithLLMProvider(&mockLLM{}))
	_, err := s.Optimize(context.Background(), "")
	if err != ErrEmptyText {
		t.Errorf("want ErrEmptyText, got %v", err)
	}
}

func TestOptimizeSuccess(t *testing.T) {
	mock := &mockLLM{
		responses: []string{
			`{"text": "I spent 3 years studying the X algorithm. Here's what nobody tells you about going viral:\n\n1. Replies are weighted 27x more than likes\n2. External links kill 70% of your reach\n\nWhat's the most surprising algorithm hack you've found?", "reasoning": "Added question hook, data points, personal story"}`,
		},
	}

	s := New(WithLLMProvider(mock))
	result, err := s.Optimize(context.Background(), "The X algorithm is interesting",
		WithMaxIterations(1),
		WithTargetScore(95),
	)

	if err == nil {
		if result.Best.Text == "" {
			t.Error("best text should not be empty")
		}
		if len(result.Iterations) != 1 {
			t.Errorf("want 1 iteration, got %d", len(result.Iterations))
		}
	} else if err != ErrMaxIterations {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOptimizeCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := New(WithLLMProvider(&mockLLM{responses: []string{`{"text":"hi","reasoning":"test"}`}}))
	_, err := s.Optimize(ctx, "test post")
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("want context canceled error, got %v", err)
	}
}

func TestOptimizeWithOptions(t *testing.T) {
	mock := &mockLLM{
		responses: []string{
			`{"text": "Optimized post for Go developers! What patterns do you use?", "reasoning": "Added question"}`,
		},
	}

	s := New(WithLLMProvider(mock))
	result, _ := s.Optimize(context.Background(), "Go is great",
		WithTone("professional"),
		WithAudience("Go developers"),
		WithConstraints("keep under 280 chars"),
		WithPreserveIntent(true),
		WithMaxIterations(1),
		WithTargetScore(99),
	)

	if result != nil && len(result.Iterations) > 0 {
		if result.Iterations[0].Text == "" {
			t.Error("iteration text should not be empty")
		}
	}
}

// === JSON extraction tests ===

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare json",
			input: `{"text": "hello", "reasoning": "test"}`,
			want:  `{"text": "hello", "reasoning": "test"}`,
		},
		{
			name:  "fenced json",
			input: "```json\n{\"text\": \"hello\"}\n```",
			want:  `{"text": "hello"}`,
		},
		{
			name:  "text before json",
			input: "Here is my response:\n{\"text\": \"hello\", \"reasoning\": \"r\"}",
			want:  `{"text": "hello", "reasoning": "r"}`,
		},
		{
			name:  "no json",
			input: "no json here",
			want:  "",
		},
		{
			name:  "nested braces",
			input: `{"text": "test {nested}", "reasoning": "ok"}`,
			want:  `{"text": "test {nested}", "reasoning": "ok"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.want {
				t.Errorf("extractJSON(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseLLMResponse(t *testing.T) {
	text, reasoning, err := parseLLMResponse(`{"text": "optimized post", "reasoning": "added question"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "optimized post" {
		t.Errorf("text = %q, want %q", text, "optimized post")
	}
	if reasoning != "added question" {
		t.Errorf("reasoning = %q, want %q", reasoning, "added question")
	}
}

func TestParseLLMResponseEmpty(t *testing.T) {
	_, _, err := parseLLMResponse("")
	if err == nil {
		t.Error("should error on empty completion")
	}
}

func TestParseLLMResponseMissingText(t *testing.T) {
	_, _, err := parseLLMResponse(`{"reasoning": "something"}`)
	if err == nil {
		t.Error("should error when text field is missing")
	}
}

// === Signal detection specifics ===

func TestSignalPersonalStory(t *testing.T) {
	s := New()
	result := s.Score("I spent 2 years building this. I learned so much about distributed systems along the way.")
	hasSignal := false
	for _, sig := range result.Signals {
		if sig.Name == "personal_story" {
			hasSignal = true
			break
		}
	}
	if !hasSignal {
		t.Error("should detect personal_story signal")
	}
}

func TestSignalControversialTake(t *testing.T) {
	s := New()
	result := s.Score("Unpopular opinion: microservices are overrated for 99% of startups. Change my mind.")
	hasSignal := false
	for _, sig := range result.Signals {
		if sig.Name == "controversial_take" {
			hasSignal = true
			break
		}
	}
	if !hasSignal {
		t.Error("should detect controversial_take signal")
	}
}

func TestSignalThreadMarker(t *testing.T) {
	s := New()
	result := s.ScorePost(Post{Text: "Here's everything I know about scaling Go services 🧵", IsThread: true})
	hasSignal := false
	for _, sig := range result.Signals {
		if sig.Name == "thread_marker" {
			hasSignal = true
			break
		}
	}
	if !hasSignal {
		t.Error("should detect thread_marker signal")
	}
}

func TestSignalExcessiveCaps(t *testing.T) {
	s := New()
	result := s.Score("THIS IS AN IMPORTANT ANNOUNCEMENT ABOUT OUR NEW PRODUCT LAUNCH TODAY")
	hasSignal := false
	for _, sig := range result.Signals {
		if sig.Name == "excessive_caps" {
			hasSignal = true
			break
		}
	}
	if !hasSignal {
		t.Error("should detect excessive_caps signal")
	}
}

func TestScoreRange(t *testing.T) {
	s := New()
	posts := []string{
		"",
		"GM",
		"Hello world",
		"What do you think?",
		"I spent a year building this tool and here's what I learned about the algorithm. What's your experience?",
	}
	for _, text := range posts {
		result := s.Score(text)
		if result.Score < 0 || result.Score > 100 {
			t.Errorf("score %d out of range [0,100] for %q", result.Score, text)
		}
	}
}

func TestLLMProviderFunc(t *testing.T) {
	var called bool
	fn := LLMProviderFunc(func(ctx context.Context, prompt string) (string, error) {
		called = true
		return `{"text": "hello", "reasoning": "test"}`, nil
	})

	s := New(WithLLMProvider(fn))
	_, _ = s.Optimize(context.Background(), "test",
		WithMaxIterations(1),
		WithTargetScore(99),
	)

	if !called {
		t.Error("LLMProviderFunc should have been called")
	}
}
