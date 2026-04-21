package viral

import (
	"math"
	"sort"
	"strings"
)

// scorePost runs the full scoring pipeline for a post with optional author
// context. The result is what callers see — score 0-100, the raw weighted
// sum, per-action probabilities, detected signals, feedback, and factor lists.
func scorePost(post Post, author *AuthorProfile, weights WeightSet, normalize func(float64) int) ScoreResult {
	if strings.TrimSpace(post.Text) == "" {
		fb := buildFeedback(post, &signalContext{}, nil, 0)
		return ScoreResult{Score: 0, Feedback: fb, NegativeFactors: []string{"Post text is empty"}}
	}

	probs := baselineActionScores()

	if post.MediaType == MediaImage {
		probs.PhotoExpand = 0.030
	}
	if post.MediaType == MediaVideo {
		probs.VideoQualityView = 0.050
	}

	ctx := newSignalContext(post)

	var detected []Signal
	for _, def := range allSignals {
		if def.detect(post, ctx) {
			def.apply(&probs)
			detected = append(detected, Signal{
				Name:        def.name,
				Category:    def.category,
				Impact:      def.impact,
				Description: def.description,
			})
		}
	}

	if author != nil {
		detected = append(detected, applyAuthorSignals(&probs, *author)...)
	}

	probs = clampProbabilities(probs)

	raw := weightedSum(probs, weights)

	score := normalize(raw)

	positive, negative := factorLists(detected)

	feedback := buildFeedback(post, ctx, detected, score)

	return ScoreResult{
		Score:           score,
		RawScore:        raw,
		ActionScores:    probs,
		Signals:         detected,
		Feedback:        feedback,
		PositiveFactors: positive,
		NegativeFactors: negative,
	}
}

// weightedSum applies the X algorithm's weighted formula:
//
//	raw = Σ (weight_i × P(action_i))
//
// Mirrors home-mixer/scorers/weighted_scorer.rs::compute_weighted_score.
func weightedSum(p ActionScores, w WeightSet) float64 {
	return p.Favorite*w.Favorite +
		p.Reply*w.Reply +
		p.Retweet*w.Retweet +
		p.Quote*w.Quote +
		p.PhotoExpand*w.PhotoExpand +
		p.Click*w.Click +
		p.ProfileClick*w.ProfileClick +
		p.VideoQualityView*w.VideoQualityView +
		p.Share*w.Share +
		p.ShareViaDM*w.ShareViaDM +
		p.ShareViaCopyLink*w.ShareViaCopyLink +
		p.Dwell*w.Dwell +
		p.DwellTime*w.DwellTime +
		p.QuotedClick*w.QuotedClick +
		p.FollowAuthor*w.FollowAuthor +
		p.NotInterested*w.NotInterested +
		p.BlockAuthor*w.BlockAuthor +
		p.MuteAuthor*w.MuteAuthor +
		p.Report*w.Report
}

// defaultNormalizer maps the raw weighted score to a 0-100 integer using a
// sigmoid calibrated so that:
//
//	raw ~= 0.1  → score ~= 20  (spam/empty)
//	raw ~= 1.5  → score = 50   (baseline)
//	raw ~= 3.0  → score ~= 82  (great)
//	raw ~= 5.0  → score ~= 97  (excellent)
//
// This roughly matches the score distribution produced by willitgoviral.xyz.
func defaultNormalizer(raw float64) int {
	const k = 1.0
	const mid = 1.5
	s := 100.0 / (1.0 + math.Exp(-k*(raw-mid)))
	if s < 0 {
		s = 0
	}
	if s > 100 {
		s = 100
	}
	return int(math.Round(s))
}

// clampProbabilities ensures probabilities stay in valid ranges. Probabilities
// are clamped to [0, 1]; DwellTime (a continuous seconds value) is clamped to
// a reasonable upper bound.
func clampProbabilities(p ActionScores) ActionScores {
	clampProb := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}
	p.Favorite = clampProb(p.Favorite)
	p.Reply = clampProb(p.Reply)
	p.Retweet = clampProb(p.Retweet)
	p.Quote = clampProb(p.Quote)
	p.Click = clampProb(p.Click)
	p.ProfileClick = clampProb(p.ProfileClick)
	p.PhotoExpand = clampProb(p.PhotoExpand)
	p.VideoQualityView = clampProb(p.VideoQualityView)
	p.Share = clampProb(p.Share)
	p.ShareViaDM = clampProb(p.ShareViaDM)
	p.ShareViaCopyLink = clampProb(p.ShareViaCopyLink)
	p.Dwell = clampProb(p.Dwell)
	p.QuotedClick = clampProb(p.QuotedClick)
	p.FollowAuthor = clampProb(p.FollowAuthor)
	p.NotInterested = clampProb(p.NotInterested)
	p.BlockAuthor = clampProb(p.BlockAuthor)
	p.MuteAuthor = clampProb(p.MuteAuthor)
	p.Report = clampProb(p.Report)

	if p.DwellTime < 0 {
		p.DwellTime = 0
	}
	if p.DwellTime > 30 {
		p.DwellTime = 30
	}
	return p
}

// factorLists splits detected signals into human-readable positive and
// negative factor lists, sorted by impact magnitude (highest first).
func factorLists(signals []Signal) (positive, negative []string) {
	type pair struct {
		desc   string
		impact float64
	}
	var pos, neg []pair

	for _, s := range signals {
		if s.Impact >= 0 {
			pos = append(pos, pair{s.Description, s.Impact})
		} else {
			neg = append(neg, pair{s.Description, -s.Impact})
		}
	}

	sort.SliceStable(pos, func(i, j int) bool { return pos[i].impact > pos[j].impact })
	sort.SliceStable(neg, func(i, j int) bool { return neg[i].impact > neg[j].impact })

	for _, p := range pos {
		positive = append(positive, p.desc)
	}
	for _, n := range neg {
		negative = append(negative, n.desc)
	}
	return positive, negative
}
