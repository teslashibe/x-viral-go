package viral

// Media is the type of media attached to a post.
type Media int

const (
	MediaNone Media = iota
	MediaImage
	MediaVideo
	MediaPoll
	MediaGIF
)

// String returns the lowercase name of the media type.
func (m Media) String() string {
	switch m {
	case MediaImage:
		return "image"
	case MediaVideo:
		return "video"
	case MediaPoll:
		return "poll"
	case MediaGIF:
		return "gif"
	default:
		return "none"
	}
}

// Post is the input to the scorer. Either Text alone (for draft scoring) or a
// fully populated Post with media/format metadata can be provided.
type Post struct {
	Text            string   `json:"text"`
	MediaType       Media    `json:"mediaType,omitempty"`
	HasExternalLink bool     `json:"hasExternalLink,omitempty"`
	IsReply         bool     `json:"isReply,omitempty"`
	IsQuote         bool     `json:"isQuote,omitempty"`
	IsThread        bool     `json:"isThread,omitempty"`
	Hashtags        []string `json:"hashtags,omitempty"`
	MentionedUsers  []string `json:"mentionedUsers,omitempty"`
}

// AuthorProfile provides optional author context that affects scoring.
// Verified/Premium accounts and accounts with strong follower-to-following
// ratios receive distribution boosts.
type AuthorProfile struct {
	FollowersCount int  `json:"followersCount"`
	FollowingCount int  `json:"followingCount"`
	IsVerified     bool `json:"isVerified"`
	IsPremium      bool `json:"isPremium"`
	TweetCount     int  `json:"tweetCount,omitempty"`
	AccountAgeDays int  `json:"accountAgeDays,omitempty"`
}

// ScoreResult is the output of scoring a single post.
type ScoreResult struct {
	Score           int          `json:"score"`
	RawScore        float64      `json:"rawScore"`
	ActionScores    ActionScores `json:"actionScores"`
	Signals         []Signal     `json:"signals"`
	Feedback        []Suggestion `json:"feedback"`
	PositiveFactors []string     `json:"positiveFactors"`
	NegativeFactors []string     `json:"negativeFactors"`
}

// ActionScores holds estimated engagement probabilities for each of the 19
// action types tracked by the X algorithm. These are heuristic estimates,
// not ML predictions.
type ActionScores struct {
	Favorite         float64 `json:"favorite"`
	Reply            float64 `json:"reply"`
	Retweet          float64 `json:"retweet"`
	PhotoExpand      float64 `json:"photoExpand"`
	Click            float64 `json:"click"`
	ProfileClick     float64 `json:"profileClick"`
	VideoQualityView float64 `json:"videoQualityView"`
	Share            float64 `json:"share"`
	ShareViaDM       float64 `json:"shareViaDM"`
	ShareViaCopyLink float64 `json:"shareViaCopyLink"`
	Dwell            float64 `json:"dwell"`
	Quote            float64 `json:"quote"`
	QuotedClick      float64 `json:"quotedClick"`
	DwellTime        float64 `json:"dwellTime"`
	FollowAuthor     float64 `json:"followAuthor"`
	NotInterested    float64 `json:"notInterested"`
	BlockAuthor      float64 `json:"blockAuthor"`
	MuteAuthor       float64 `json:"muteAuthor"`
	Report           float64 `json:"report"`
}

// SignalCategory groups signals by what they affect.
type SignalCategory string

const (
	CategoryEngagement SignalCategory = "engagement"
	CategoryFormat     SignalCategory = "format"
	CategoryNegative   SignalCategory = "negative"
	CategoryAuthor     SignalCategory = "author"
)

// Signal is a detected content signal that affects scoring.
type Signal struct {
	Name        string         `json:"name"`
	Category    SignalCategory `json:"category"`
	Impact      float64        `json:"impact"`
	Description string         `json:"description"`
}

// Suggestion is a specific, actionable improvement recommendation.
type Suggestion struct {
	Priority    int    `json:"priority"`
	Category    string `json:"category"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

// OptimizeResult is the output of LLM-assisted optimization.
type OptimizeResult struct {
	Original    ScoreResult `json:"original"`
	Iterations  []Iteration `json:"iterations"`
	Best        Iteration   `json:"best"`
	TotalTokens int         `json:"totalTokens,omitempty"`
}

// Iteration represents one LLM rewrite attempt.
type Iteration struct {
	Number    int         `json:"number"`
	Text      string      `json:"text"`
	Score     ScoreResult `json:"score"`
	Reasoning string      `json:"reasoning"`
}
