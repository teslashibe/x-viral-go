package viral


// Scorer evaluates posts against the X algorithm's weighted scoring formula.
//
// Scorer is safe for concurrent use. Score, ScorePost, ScoreWithAuthor, and
// ScoreBatch are pure functions with no side effects.
type Scorer struct {
	weights    WeightSet
	provider   LLMProvider
	normalizer func(float64) int
}

// Option configures a Scorer.
type Option func(*Scorer)

// New creates a Scorer with default weights and configuration.
//
// No auth or network access is required for scoring — Score and ScorePost
// are pure local computation. An LLMProvider is only required if Optimize
// or OptimizePost will be called.
func New(opts ...Option) *Scorer {
	s := &Scorer{
		weights:    DefaultWeights(),
		normalizer: defaultNormalizer,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// WithWeights overrides the default action weights. Callers can tune these
// to reflect observed algorithm changes or domain-specific scoring.
func WithWeights(w WeightSet) Option {
	return func(s *Scorer) { s.weights = w }
}

// WithLLMProvider sets the LLM backend used by Optimize and OptimizePost.
// Without a provider, those methods return ErrNoLLMProvider.
func WithLLMProvider(p LLMProvider) Option {
	return func(s *Scorer) { s.provider = p }
}

// WithScoreNormalizer overrides the default 0-100 score normalization
// function. The default uses a sigmoid calibrated against typical raw scores.
func WithScoreNormalizer(fn func(float64) int) Option {
	return func(s *Scorer) {
		if fn != nil {
			s.normalizer = fn
		}
	}
}

// Weights returns a copy of the scorer's current weight configuration.
func (s *Scorer) Weights() WeightSet {
	return s.weights
}

// Score evaluates a plain text post and returns a viral score (0-100) with
// detected signals, action probability estimates, and feedback.
//
// For posts with media, threads, or other format metadata, use ScorePost.
func (s *Scorer) Score(text string) ScoreResult {
	post := buildPostFromText(text)
	return scorePost(post, nil, s.weights, s.normalizer)
}

// ScorePost evaluates a structured Post (with media type, format flags, etc.)
// and returns a viral score (0-100) with detected signals, action probability
// estimates, and feedback.
func (s *Scorer) ScorePost(post Post) ScoreResult {
	enrichPost(&post)
	return scorePost(post, nil, s.weights, s.normalizer)
}

// ScoreWithAuthor evaluates a structured Post with author profile context.
// Verified/Premium accounts and accounts with strong follower-to-following
// ratios receive distribution boosts that affect the final score.
func (s *Scorer) ScoreWithAuthor(post Post, author AuthorProfile) ScoreResult {
	enrichPost(&post)
	return scorePost(post, &author, s.weights, s.normalizer)
}

// ScoreBatch scores multiple posts independently and returns the results in
// the same order. Useful for comparing draft variations side-by-side.
func (s *Scorer) ScoreBatch(posts []Post) []ScoreResult {
	results := make([]ScoreResult, len(posts))
	for i, p := range posts {
		enrichPost(&p)
		results[i] = scorePost(p, nil, s.weights, s.normalizer)
	}
	return results
}

// buildPostFromText constructs a Post from raw text, auto-detecting media-free
// signals (external links, hashtags, mentions, thread markers).
func buildPostFromText(text string) Post {
	post := Post{Text: text}
	enrichPost(&post)
	return post
}

// enrichPost auto-fills detectable Post fields from the text body.
// It does not overwrite fields the caller has explicitly set.
func enrichPost(post *Post) {
	if !post.HasExternalLink {
		post.HasExternalLink = urlRegex.MatchString(post.Text)
	}
	if len(post.Hashtags) == 0 {
		post.Hashtags = extractHashtags(post.Text)
	}
	if len(post.MentionedUsers) == 0 {
		post.MentionedUsers = extractMentions(post.Text)
	}
	if !post.IsThread {
		post.IsThread = threadRegex.MatchString(post.Text)
	}
}

func extractHashtags(text string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(text, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			out = append(out, m[1])
		}
	}
	return out
}

func extractMentions(text string) []string {
	matches := mentionRegex.FindAllStringSubmatch(text, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			out = append(out, m[1])
		}
	}
	return out
}

