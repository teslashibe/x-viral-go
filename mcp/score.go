package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/teslashibe/mcptool"
	viral "github.com/teslashibe/x-viral-go"
)

// PostInput is an MCP-friendly representation of [viral.Post]. MediaType is
// expressed as a lowercase string (none|image|video|poll|gif) instead of the
// underlying integer enum.
type PostInput struct {
	Text            string   `json:"text" jsonschema:"description=raw post text to score (required if no other fields imply text),required"`
	MediaType       string   `json:"media_type,omitempty" jsonschema:"description=attached media type; allowed: none,image,video,poll,gif,enum=none,enum=image,enum=video,enum=poll,enum=gif,default=none"`
	HasExternalLink bool     `json:"has_external_link,omitempty" jsonschema:"description=whether the post links off-platform (X applies a 40-70% distribution penalty)"`
	IsReply         bool     `json:"is_reply,omitempty" jsonschema:"description=true if this post is a reply to another post"`
	IsQuote         bool     `json:"is_quote,omitempty" jsonschema:"description=true if this post quotes another post"`
	IsThread        bool     `json:"is_thread,omitempty" jsonschema:"description=true if this post is part of a thread (also auto-detected from text)"`
	Hashtags        []string `json:"hashtags,omitempty" jsonschema:"description=hashtags without the # prefix (auto-extracted from text if omitted)"`
	MentionedUsers  []string `json:"mentioned_users,omitempty" jsonschema:"description=mentioned handles without the @ prefix (auto-extracted from text if omitted)"`
}

// AuthorInput is the MCP input shape mirroring [viral.AuthorProfile].
type AuthorInput struct {
	FollowersCount int  `json:"followers_count,omitempty" jsonschema:"description=author's follower count,minimum=0"`
	FollowingCount int  `json:"following_count,omitempty" jsonschema:"description=accounts the author follows,minimum=0"`
	IsVerified     bool `json:"is_verified,omitempty" jsonschema:"description=true if the author has the legacy verified badge"`
	IsPremium      bool `json:"is_premium,omitempty" jsonschema:"description=true if the author has X Premium (paid blue check)"`
	TweetCount     int  `json:"tweet_count,omitempty" jsonschema:"description=lifetime post count for the author,minimum=0"`
	AccountAgeDays int  `json:"account_age_days,omitempty" jsonschema:"description=account age in days,minimum=0"`
}

// ScoreTextInput is the typed input for xviral_score_text.
type ScoreTextInput struct {
	Text string `json:"text" jsonschema:"description=raw post text to score for viral potential (0-100),required"`
}

// ScorePostInput is the typed input for xviral_score_post.
type ScorePostInput struct {
	Post PostInput `json:"post" jsonschema:"description=structured post payload with media/format metadata,required"`
}

// ScoreWithAuthorInput is the typed input for xviral_score_with_author.
type ScoreWithAuthorInput struct {
	Post   PostInput   `json:"post" jsonschema:"description=structured post payload with media/format metadata,required"`
	Author AuthorInput `json:"author" jsonschema:"description=author profile context (verified/premium/follower ratio influence the score),required"`
}

// ScoreBatchInput is the typed input for xviral_score_batch.
type ScoreBatchInput struct {
	Posts []PostInput `json:"posts" jsonschema:"description=2-50 draft variations to score side-by-side,required,minItems=1,maxItems=50"`
}

func scoreText(_ context.Context, s *viral.Scorer, in ScoreTextInput) (any, error) {
	if strings.TrimSpace(in.Text) == "" {
		return nil, &mcptool.Error{Code: "invalid_input", Message: "text must not be empty"}
	}
	return s.Score(in.Text), nil
}

func scoreStructuredPost(_ context.Context, s *viral.Scorer, in ScorePostInput) (any, error) {
	post, err := toViralPost(in.Post)
	if err != nil {
		return nil, err
	}
	return s.ScorePost(post), nil
}

func scoreWithAuthor(_ context.Context, s *viral.Scorer, in ScoreWithAuthorInput) (any, error) {
	post, err := toViralPost(in.Post)
	if err != nil {
		return nil, err
	}
	return s.ScoreWithAuthor(post, toViralAuthor(in.Author)), nil
}

func scoreBatch(_ context.Context, s *viral.Scorer, in ScoreBatchInput) (any, error) {
	if len(in.Posts) == 0 {
		return nil, &mcptool.Error{Code: "invalid_input", Message: "posts must contain at least one item"}
	}
	posts := make([]viral.Post, 0, len(in.Posts))
	for i, p := range in.Posts {
		post, err := toViralPost(p)
		if err != nil {
			return nil, &mcptool.Error{Code: "invalid_input", Message: fmt.Sprintf("posts[%d]: %v", i, err)}
		}
		posts = append(posts, post)
	}
	results := s.ScoreBatch(posts)
	return mcptool.PageOf(results, "", len(results)), nil
}

func toViralPost(in PostInput) (viral.Post, error) {
	media, err := parseMedia(in.MediaType)
	if err != nil {
		return viral.Post{}, err
	}
	return viral.Post{
		Text:            in.Text,
		MediaType:       media,
		HasExternalLink: in.HasExternalLink,
		IsReply:         in.IsReply,
		IsQuote:         in.IsQuote,
		IsThread:        in.IsThread,
		Hashtags:        in.Hashtags,
		MentionedUsers:  in.MentionedUsers,
	}, nil
}

func toViralAuthor(in AuthorInput) viral.AuthorProfile {
	return viral.AuthorProfile{
		FollowersCount: in.FollowersCount,
		FollowingCount: in.FollowingCount,
		IsVerified:     in.IsVerified,
		IsPremium:      in.IsPremium,
		TweetCount:     in.TweetCount,
		AccountAgeDays: in.AccountAgeDays,
	}
}

func parseMedia(s string) (viral.Media, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "none":
		return viral.MediaNone, nil
	case "image":
		return viral.MediaImage, nil
	case "video":
		return viral.MediaVideo, nil
	case "poll":
		return viral.MediaPoll, nil
	case "gif":
		return viral.MediaGIF, nil
	default:
		return viral.MediaNone, &mcptool.Error{
			Code:    "invalid_input",
			Message: fmt.Sprintf("media_type %q invalid; allowed: none,image,video,poll,gif", s),
		}
	}
}

var scoreTools = []mcptool.Tool{
	mcptool.Define[*viral.Scorer, ScoreTextInput](
		"xviral_score_text",
		"Score plain X (Twitter) post text 0-100 with detected signals, action probabilities, and feedback",
		"Score",
		scoreText,
	),
	mcptool.Define[*viral.Scorer, ScorePostInput](
		"xviral_score_post",
		"Score a structured post (text + media type + format flags) for X algorithm viral potential",
		"ScorePost",
		scoreStructuredPost,
	),
	mcptool.Define[*viral.Scorer, ScoreWithAuthorInput](
		"xviral_score_with_author",
		"Score a post with author profile context (verified/premium/follower ratio shift the final score)",
		"ScoreWithAuthor",
		scoreWithAuthor,
	),
	mcptool.Define[*viral.Scorer, ScoreBatchInput](
		"xviral_score_batch",
		"Score multiple draft post variations independently for side-by-side comparison",
		"ScoreBatch",
		scoreBatch,
	),
}
