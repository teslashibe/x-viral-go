package viral

import "sort"

// buildFeedback turns detected signals into actionable Suggestions for
// improving the post. Negative signals always produce fix-it suggestions;
// missing high-impact positive signals produce add-it suggestions.
func buildFeedback(post Post, ctx *signalContext, signals []Signal, score int) []Suggestion {
	detected := make(map[string]Signal, len(signals))
	for _, s := range signals {
		detected[s.Name] = s
	}

	var suggestions []Suggestion

	for _, rule := range feedbackRules {
		if s := rule(post, ctx, detected, score); s != nil {
			suggestions = append(suggestions, *s)
		}
	}

	sort.SliceStable(suggestions, func(i, j int) bool {
		return suggestions[i].Priority < suggestions[j].Priority
	})

	return suggestions
}

// feedbackRule is a function that returns a Suggestion if it applies to the
// given post, or nil if it doesn't.
type feedbackRule func(post Post, ctx *signalContext, detected map[string]Signal, score int) *Suggestion

var feedbackRules = []feedbackRule{

	// === Critical issues (fix these first) ===
	rule("spam_patterns", 1, "remove", "high",
		"Remove engagement-bait phrases (\"follow for follow\", \"like and retweet\", etc.). They trigger reports and crater distribution.",
		negativeFix),

	rule("external_link", 1, "remove", "high",
		"Remove the external link or move it to a reply. External links trigger a 40-70% distribution penalty on X — the algorithm strongly prefers keeping users on-platform.",
		negativeFix),

	rule("excessive_caps", 2, "rewrite", "high",
		"Reduce uppercase usage. Posts with >30% caps read as shouting and trigger \"not interested\" / mute signals.",
		negativeFix),

	rule("hashtag_stuffing", 2, "remove", "medium",
		"Cut down to 0-3 hashtags. Hashtag stuffing reduces NLP relevance and looks spammy.",
		negativeFix),

	rule("excessive_emojis", 3, "rewrite", "medium",
		"Trim emoji usage. >5 emojis hurts readability and can trigger \"not interested\" signals.",
		negativeFix),

	rule("mention_heavy", 3, "remove", "medium",
		"Reduce @mentions to 1-2. More than 3 mentions can read as tag-farming and increase mute rate.",
		negativeFix),

	rule("clickbait", 3, "rewrite", "medium",
		"Soften clickbait phrasing. It boosts clicks short-term but raises mute / not-interested rates that hurt your account long-term.",
		negativeFix),

	rule("empty_calories", 2, "rewrite", "high",
		"Add substance. Generic posts (\"GM\", \"this!\", motivational quotes) get little engagement and don't build audience.",
		negativeFix),

	// === High-impact additions (biggest score lifts) ===
	missingRule("question_hook", 4, "add", "high",
		"Add a question. Replies are weighted ~27x more than likes — the single highest-impact change you can make."),

	missingRule("personal_story", 5, "add", "high",
		"Open with a personal story or first-person experience (\"I learned...\", \"I just shipped...\"). Drives replies, favorites, and follows."),

	missingRule("data_point", 6, "add", "medium",
		"Include a specific number, percentage, or money figure. Concrete data adds credibility and boosts retweets/quotes."),

	missingRule("controversial_take", 7, "rewrite", "medium",
		"Take a clear position or hot take. Controversial-but-defensible opinions drive replies and quote-tweets."),

	missingRule("call_to_action", 8, "add", "medium",
		"End with a specific call-to-action (\"What's your take?\", \"Reply with your experience\")."),

	// === Format improvements ===
	missingRule("thread_marker", 9, "add", "low",
		"If this is a thread opener, add a thread indicator (a 🧵 or \"1/\"). Thread markers boost dwell time and click-through."),

	missingRule("listicle_format", 10, "restructure", "low",
		"Consider breaking into a numbered list. Listicles consistently outperform paragraphs on dwell time and shares."),

	missingRule("whitespace_structure", 10, "restructure", "low",
		"Add line breaks to create visual hierarchy. Wall-of-text reduces dwell time."),

	// === Length issues ===
	func(post Post, ctx *signalContext, detected map[string]Signal, score int) *Suggestion {
		if ctx.charCount > 0 && ctx.charCount < 50 {
			return &Suggestion{
				Priority:    4,
				Category:    "format",
				Action:      "expand",
				Description: "Post is very short. Add context, a story, or supporting detail — short posts rarely accumulate dwell time.",
				Impact:      "high",
			}
		}
		return nil
	},

	// === Media suggestions ===
	func(post Post, ctx *signalContext, detected map[string]Signal, score int) *Suggestion {
		if post.MediaType == MediaNone && score < 70 && ctx.charCount > 100 {
			return &Suggestion{
				Priority:    11,
				Category:    "format",
				Action:      "add",
				Description: "Add an image or video. Native media gets 2-10x the engagement of text-only posts.",
				Impact:      "medium",
			}
		}
		return nil
	},
}

// rule produces a feedback rule that triggers when the given negative signal
// is detected.
func rule(signalName string, priority int, action, impact, description string, kind ruleKind) feedbackRule {
	return func(post Post, ctx *signalContext, detected map[string]Signal, score int) *Suggestion {
		_, present := detected[signalName]
		switch kind {
		case negativeFix:
			if !present {
				return nil
			}
		case missingPositive:
			if present {
				return nil
			}
		}
		return &Suggestion{
			Priority:    priority,
			Category:    string(detected[signalName].Category),
			Action:      action,
			Description: description,
			Impact:      impact,
		}
	}
}

// missingRule produces a feedback rule that triggers when the given positive
// signal is NOT detected.
func missingRule(signalName string, priority int, action, impact, description string) feedbackRule {
	return func(post Post, ctx *signalContext, detected map[string]Signal, score int) *Suggestion {
		if _, present := detected[signalName]; present {
			return nil
		}
		return &Suggestion{
			Priority:    priority,
			Category:    "engagement",
			Action:      action,
			Description: description,
			Impact:      impact,
		}
	}
}

type ruleKind int

const (
	negativeFix ruleKind = iota
	missingPositive
)
