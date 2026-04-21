package viral

import (
	"regexp"
	"strings"
	"unicode"
)

// signalContext caches text features computed once per scoring run.
type signalContext struct {
	text             string
	lower            string
	wordCount        int
	charCount        int
	sentenceCount    int
	paragraphCount   int
	hashtagCount     int
	mentionCount     int
	emojiCount       int
	urls             []string
	hasURL           bool
	upperRatio       float64
	firstSentence    string
	endsWithQuestion bool
}

var (
	urlRegex      = regexp.MustCompile(`https?://[^\s]+`)
	hashtagRegex  = regexp.MustCompile(`(?:^|\s)#(\w+)`)
	mentionRegex  = regexp.MustCompile(`(?:^|\s)@(\w+)`)
	numberRegex   = regexp.MustCompile(`\b\d+(\.\d+)?%?\b`)
	listRegex     = regexp.MustCompile(`(?m)^\s*\d+[.)\]]\s+`)
	bulletRegex   = regexp.MustCompile(`(?m)^\s*[-*•]\s+`)
	threadRegex   = regexp.MustCompile(`(?i)(thread:|🧵|^\s*1\s*/|a thread\b)`)
	moneyRegex    = regexp.MustCompile(`\$[\d,]+`)
	sentenceSplit = regexp.MustCompile(`[.!?]+\s+`)
)

// newSignalContext computes derived features from a post.
func newSignalContext(post Post) *signalContext {
	text := post.Text
	lower := strings.ToLower(text)
	c := &signalContext{
		text:      text,
		lower:     lower,
		charCount: len([]rune(text)),
	}

	c.wordCount = len(strings.Fields(text))

	if sentences := sentenceSplit.Split(strings.TrimSpace(text), -1); len(sentences) > 0 {
		c.sentenceCount = 0
		for _, s := range sentences {
			if strings.TrimSpace(s) != "" {
				c.sentenceCount++
			}
		}
		c.firstSentence = strings.TrimSpace(sentences[0])
	}
	if c.sentenceCount == 0 && c.charCount > 0 {
		c.sentenceCount = 1
		c.firstSentence = text
	}

	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			c.paragraphCount++
		}
	}

	c.urls = urlRegex.FindAllString(text, -1)
	c.hasURL = len(c.urls) > 0

	c.hashtagCount = len(hashtagRegex.FindAllString(text, -1))
	c.mentionCount = len(mentionRegex.FindAllString(text, -1))
	c.emojiCount = countEmojis(text)

	c.upperRatio = upperRatio(text)

	trimmed := strings.TrimSpace(text)
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '?' {
		c.endsWithQuestion = true
	}

	return c
}

// signalDef defines a single content signal.
type signalDef struct {
	name        string
	category    SignalCategory
	description string
	impact      float64
	detect      func(p Post, c *signalContext) bool
	apply       func(probs *ActionScores)
}

// allSignals is the complete catalog of signals viral-go detects.
var allSignals = []signalDef{
	// === Engagement triggers (positive) ===
	{
		name:        "question_hook",
		category:    CategoryEngagement,
		description: "Post asks a question — directly invites replies (the highest-weight action)",
		impact:      8.0,
		detect: func(p Post, c *signalContext) bool {
			return c.endsWithQuestion || containsAny(c.lower,
				"what do you think", "what's your", "thoughts?",
				"any one else", "anyone else", "agree?", "thoughts on",
				"have you ever", "what would you", "how do you")
		},
		apply: func(probs *ActionScores) {
			probs.Reply *= 5.0
			probs.Favorite *= 1.2
			probs.Quote *= 1.3
		},
	},
	{
		name:        "call_to_action",
		category:    CategoryEngagement,
		description: "Explicit call-to-action invites engagement",
		impact:      3.0,
		detect: func(p Post, c *signalContext) bool {
			return containsAny(c.lower,
				"let me know", "share if", "rt if you", "drop a", "leave a",
				"tell me", "comment below", "reply with", "tag someone",
				"like this if", "raise your hand")
		},
		apply: func(probs *ActionScores) {
			probs.Reply *= 2.5
			probs.Quote *= 1.4
			probs.Share *= 1.3
		},
	},
	{
		name:        "controversial_take",
		category:    CategoryEngagement,
		description: "Hot take or contrarian opinion — drives debate and quote tweets",
		impact:      6.0,
		detect: func(p Post, c *signalContext) bool {
			return containsAny(c.lower,
				"unpopular opinion", "hot take", "controversial",
				"hot take:", "no one is talking about", "nobody talks about",
				"is overrated", "is underrated", "change my mind",
				"i'll die on this hill", "fight me")
		},
		apply: func(probs *ActionScores) {
			probs.Reply *= 4.0
			probs.Quote *= 3.0
			probs.Retweet *= 1.5
			probs.NotInterested *= 1.2
		},
	},
	{
		name:        "personal_story",
		category:    CategoryEngagement,
		description: "First-person narrative or experience — builds connection",
		impact:      4.0,
		detect: func(p Post, c *signalContext) bool {
			return containsAny(c.lower,
				"i learned", "my experience", "i just", "i spent",
				"i tried", "i built", "i shipped", "i discovered",
				"the other day i", "yesterday i", "last week i",
				"a few years ago", "when i was", "i used to")
		},
		apply: func(probs *ActionScores) {
			probs.Reply *= 2.5
			probs.Favorite *= 2.0
			probs.Dwell *= 1.3
			probs.FollowAuthor *= 1.5
		},
	},
	{
		name:        "data_point",
		category:    CategoryEngagement,
		description: "Specific numbers, percentages, or money figures add credibility",
		impact:      3.0,
		detect: func(p Post, c *signalContext) bool {
			return numberRegex.MatchString(c.text) || moneyRegex.MatchString(c.text)
		},
		apply: func(probs *ActionScores) {
			probs.Retweet *= 1.5
			probs.Quote *= 1.4
			probs.Share *= 1.5
			probs.ShareViaCopyLink *= 1.3
		},
	},
	{
		name:        "strong_opener",
		category:    CategoryEngagement,
		description: "Punchy first sentence acts as a hook",
		impact:      2.0,
		detect: func(p Post, c *signalContext) bool {
			fs := c.firstSentence
			if fs == "" {
				return false
			}
			n := len([]rune(fs))
			return n >= 8 && n <= 80 && c.wordCount > 5
		},
		apply: func(probs *ActionScores) {
			probs.Click *= 1.5
			probs.Dwell *= 1.3
			probs.DwellTime *= 1.4
		},
	},
	{
		name:        "vulnerability",
		category:    CategoryEngagement,
		description: "Honest or vulnerable language drives high-quality engagement",
		impact:      3.0,
		detect: func(p Post, c *signalContext) bool {
			return containsAny(c.lower,
				"honest", "honestly", "afraid", "scared", "struggle",
				"failed", "embarrassing", "vulnerable", "imposter",
				"don't know", "don't understand")
		},
		apply: func(probs *ActionScores) {
			probs.Reply *= 2.0
			probs.Favorite *= 1.5
			probs.FollowAuthor *= 1.5
		},
	},

	// === Format (positive) ===
	{
		name:        "thread_marker",
		category:    CategoryFormat,
		description: "Thread indicator boosts dwell time and click-through",
		impact:      4.0,
		detect: func(p Post, c *signalContext) bool {
			return p.IsThread || threadRegex.MatchString(c.text)
		},
		apply: func(probs *ActionScores) {
			probs.DwellTime *= 2.0
			probs.Click *= 1.5
			probs.Dwell *= 1.3
		},
	},
	{
		name:        "whitespace_structure",
		category:    CategoryFormat,
		description: "Visual hierarchy from line breaks improves scannability",
		impact:      1.5,
		detect: func(p Post, c *signalContext) bool {
			return c.paragraphCount >= 1 && c.charCount >= 140
		},
		apply: func(probs *ActionScores) {
			probs.Dwell *= 1.2
			probs.DwellTime *= 1.3
		},
	},
	{
		name:        "optimal_length_short",
		category:    CategoryFormat,
		description: "Punchy length (100-280 chars) maximizes shareability",
		impact:      1.5,
		detect: func(p Post, c *signalContext) bool {
			return c.charCount >= 100 && c.charCount <= 280
		},
		apply: func(probs *ActionScores) {
			probs.Retweet *= 1.3
			probs.Share *= 1.2
			probs.ShareViaCopyLink *= 1.2
		},
	},
	{
		name:        "optimal_length_long",
		category:    CategoryFormat,
		description: "Long-form (>800 chars) maximizes dwell time",
		impact:      2.5,
		detect: func(p Post, c *signalContext) bool {
			return c.charCount > 800
		},
		apply: func(probs *ActionScores) {
			probs.DwellTime *= 2.0
			probs.Dwell *= 1.5
		},
	},
	{
		name:        "listicle_format",
		category:    CategoryFormat,
		description: "Numbered or bulleted lists boost dwell and shareability",
		impact:      2.5,
		detect: func(p Post, c *signalContext) bool {
			lists := len(listRegex.FindAllString(c.text, -1))
			bullets := len(bulletRegex.FindAllString(c.text, -1))
			return lists+bullets >= 2
		},
		apply: func(probs *ActionScores) {
			probs.Dwell *= 1.5
			probs.DwellTime *= 1.5
			probs.Share *= 1.3
			probs.ShareViaCopyLink *= 1.3
		},
	},
	{
		name:        "has_image",
		category:    CategoryFormat,
		description: "Images get 2-3x engagement vs text-only",
		impact:      3.0,
		detect: func(p Post, c *signalContext) bool {
			return p.MediaType == MediaImage
		},
		apply: func(probs *ActionScores) {
			probs.PhotoExpand = 0.030
			probs.Favorite *= 1.5
			probs.Retweet *= 1.3
		},
	},
	{
		name:        "has_video",
		category:    CategoryFormat,
		description: "Native video gets ~10x engagement vs text-only",
		impact:      5.0,
		detect: func(p Post, c *signalContext) bool {
			return p.MediaType == MediaVideo
		},
		apply: func(probs *ActionScores) {
			probs.VideoQualityView = 0.050
			probs.Favorite *= 1.8
			probs.Retweet *= 1.5
			probs.Dwell *= 1.5
			probs.DwellTime *= 2.0
		},
	},
	{
		name:        "has_poll",
		category:    CategoryFormat,
		description: "Polls explicitly invite a reply-equivalent action",
		impact:      3.0,
		detect: func(p Post, c *signalContext) bool {
			return p.MediaType == MediaPoll
		},
		apply: func(probs *ActionScores) {
			probs.Reply *= 2.0
			probs.Dwell *= 1.3
		},
	},

	// === Negative signals ===
	{
		name:        "external_link",
		category:    CategoryNegative,
		description: "External links trigger a 40-70% distribution penalty (X keeps users on-platform)",
		impact:      -8.0,
		detect: func(p Post, c *signalContext) bool {
			return p.HasExternalLink || c.hasURL
		},
		apply: func(probs *ActionScores) {
			multiplyPositives(probs, 0.4)
			probs.NotInterested *= 1.3
		},
	},
	{
		name:        "hashtag_stuffing",
		category:    CategoryNegative,
		description: "More than 3 hashtags reduces NLP relevance and looks spammy",
		impact:      -2.0,
		detect: func(p Post, c *signalContext) bool {
			n := c.hashtagCount + len(p.Hashtags)
			return n > 3
		},
		apply: func(probs *ActionScores) {
			multiplyPositives(probs, 0.7)
			probs.NotInterested *= 1.3
		},
	},
	{
		name:        "excessive_caps",
		category:    CategoryNegative,
		description: "More than 30% uppercase reads as shouting and triggers mute/not-interested",
		impact:      -3.0,
		detect: func(p Post, c *signalContext) bool {
			return c.upperRatio > 0.30 && c.charCount > 20
		},
		apply: func(probs *ActionScores) {
			probs.NotInterested *= 3.0
			probs.MuteAuthor *= 2.0
			probs.Favorite *= 0.7
		},
	},
	{
		name:        "excessive_emojis",
		category:    CategoryNegative,
		description: "Too many emojis (>5 or high density) hurts readability and engagement",
		impact:      -2.0,
		detect: func(p Post, c *signalContext) bool {
			if c.emojiCount > 5 {
				return true
			}
			if c.charCount > 0 && float64(c.emojiCount)/float64(c.charCount) > 0.20 {
				return true
			}
			return false
		},
		apply: func(probs *ActionScores) {
			probs.NotInterested *= 2.0
			probs.Favorite *= 0.8
		},
	},
	{
		name:        "spam_patterns",
		category:    CategoryNegative,
		description: "Engagement bait phrases trigger reports, blocks, and reduced distribution",
		impact:      -10.0,
		detect: func(p Post, c *signalContext) bool {
			return containsAny(c.lower,
				"follow for follow", "f4f", "follow back", "follow me",
				"like and retweet", "rt for", "drop a follow",
				"first to comment", "money in the dms")
		},
		apply: func(probs *ActionScores) {
			probs.Report *= 5.0
			probs.BlockAuthor *= 3.0
			probs.NotInterested *= 3.0
			multiplyPositives(probs, 0.5)
		},
	},
	{
		name:        "mention_heavy",
		category:    CategoryNegative,
		description: "More than 3 @mentions can read as spam or tag-farming",
		impact:      -1.5,
		detect: func(p Post, c *signalContext) bool {
			n := c.mentionCount + len(p.MentionedUsers)
			return n > 3
		},
		apply: func(probs *ActionScores) {
			probs.MuteAuthor *= 1.5
			probs.NotInterested *= 1.3
			multiplyPositives(probs, 0.85)
		},
	},
	{
		name:        "empty_calories",
		category:    CategoryNegative,
		description: "Generic/low-substance content gets little engagement",
		impact:      -3.0,
		detect: func(p Post, c *signalContext) bool {
			t := strings.TrimSpace(c.lower)
			if t == "gm" || t == "gn" || t == "this" || t == "this!" || t == "+1" {
				return true
			}
			if c.wordCount < 3 && c.charCount < 20 {
				return true
			}
			return containsAny(t,
				"this is the way", "based", "ngmi", "wagmi",
				"hot take coming", "retweet this if you agree")
		},
		apply: func(probs *ActionScores) {
			multiplyPositives(probs, 0.5)
			probs.Reply *= 0.5
		},
	},
	{
		name:        "clickbait",
		category:    CategoryNegative,
		description: "Clickbait phrasing draws clicks but raises not-interested and mutes",
		impact:      -1.5,
		detect: func(p Post, c *signalContext) bool {
			return containsAny(c.lower,
				"you won't believe", "this changed my life",
				"the secret to", "they don't want you to know",
				"shocking truth", "what they don't tell you")
		},
		apply: func(probs *ActionScores) {
			probs.NotInterested *= 1.5
			probs.Click *= 1.3
			probs.Favorite *= 0.7
		},
	},
}

// applyAuthorSignals adjusts probabilities based on author profile.
// Returns the signals that were applied for inclusion in the result.
func applyAuthorSignals(probs *ActionScores, author AuthorProfile) []Signal {
	var signals []Signal

	if author.IsPremium {
		multiplyPositives(probs, 1.5)
		signals = append(signals, Signal{
			Name:        "premium_boost",
			Category:    CategoryAuthor,
			Impact:      5.0,
			Description: "X Premium subscribers get 2-4x distribution boost",
		})
	}

	if author.IsVerified && !author.IsPremium {
		multiplyPositives(probs, 1.2)
		signals = append(signals, Signal{
			Name:        "verified_authority",
			Category:    CategoryAuthor,
			Impact:      2.0,
			Description: "Verified accounts carry higher baseline trust",
		})
	}

	if author.FollowingCount > 0 {
		ratio := float64(author.FollowersCount) / float64(author.FollowingCount)
		switch {
		case ratio >= 10:
			multiplyPositives(probs, 1.3)
			signals = append(signals, Signal{
				Name:        "high_authority_ratio",
				Category:    CategoryAuthor,
				Impact:      3.0,
				Description: "Strong follower-to-following ratio (>=10:1) signals authority",
			})
		case ratio >= 2:
			multiplyPositives(probs, 1.15)
			signals = append(signals, Signal{
				Name:        "authority_ratio",
				Category:    CategoryAuthor,
				Impact:      1.5,
				Description: "Healthy follower-to-following ratio (>=2:1)",
			})
		case ratio < 0.5:
			multiplyPositives(probs, 0.85)
			signals = append(signals, Signal{
				Name:        "weak_authority_ratio",
				Category:    CategoryAuthor,
				Impact:      -1.5,
				Description: "Low follower-to-following ratio reduces baseline authority",
			})
		}
	}

	if author.AccountAgeDays > 0 && author.AccountAgeDays < 30 {
		multiplyPositives(probs, 0.7)
		signals = append(signals, Signal{
			Name:        "new_account",
			Category:    CategoryAuthor,
			Impact:      -3.0,
			Description: "Accounts under 30 days old get reduced distribution",
		})
	}

	return signals
}

// === Helpers ===

// baselineActionScores returns the per-impression action probabilities for a
// neutral post (no detected signals). These approximate empirical engagement
// rates from public X data.
func baselineActionScores() ActionScores {
	return ActionScores{
		Favorite:         0.020,
		Reply:            0.001,
		Retweet:          0.005,
		Quote:            0.0005,
		Click:            0.030,
		ProfileClick:     0.005,
		PhotoExpand:      0.0,
		VideoQualityView: 0.0,
		Share:            0.005,
		ShareViaDM:       0.002,
		ShareViaCopyLink: 0.003,
		Dwell:            0.30,
		DwellTime:        1.0,
		QuotedClick:      0.0,
		FollowAuthor:     0.001,
		NotInterested:    0.001,
		BlockAuthor:      0.0001,
		MuteAuthor:       0.0002,
		Report:           0.00005,
	}
}

func multiplyPositives(probs *ActionScores, factor float64) {
	probs.Favorite *= factor
	probs.Reply *= factor
	probs.Retweet *= factor
	probs.Quote *= factor
	probs.Click *= factor
	probs.ProfileClick *= factor
	probs.PhotoExpand *= factor
	probs.VideoQualityView *= factor
	probs.Share *= factor
	probs.ShareViaDM *= factor
	probs.ShareViaCopyLink *= factor
	probs.Dwell *= factor
	probs.DwellTime *= factor
	probs.QuotedClick *= factor
	probs.FollowAuthor *= factor
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func countEmojis(s string) int {
	n := 0
	for _, r := range s {
		switch {
		case r >= 0x1F300 && r <= 0x1FAFF:
			n++
		case r >= 0x2600 && r <= 0x27BF:
			n++
		case r >= 0x1F000 && r <= 0x1F0FF:
			n++
		case r == 0x2728 || r == 0x2B50 || r == 0x2B55:
			n++
		}
	}
	return n
}

func upperRatio(s string) float64 {
	letters := 0
	upper := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	if letters == 0 {
		return 0
	}
	return float64(upper) / float64(letters)
}
