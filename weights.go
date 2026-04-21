package viral

// WeightSet defines the multiplicative weights for each action in the scoring
// formula. These mirror the structure in xai-org/x-algorithm's weighted_scorer.rs.
// The actual production values are not published; defaults are best-estimate
// approximations from public algorithm research.
type WeightSet struct {
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

// DefaultWeights returns the best-estimate weights derived from public analysis
// of X's algorithm. Reply is weighted ~27x a like; quote ~13x; retweet ~4x;
// negative actions (block/mute/report) carry strong negative weights.
func DefaultWeights() WeightSet {
	return WeightSet{
		Favorite:         1.0,
		Reply:            27.0,
		Retweet:          4.0,
		Quote:            13.0,
		Share:            3.0,
		ShareViaDM:       2.5,
		ShareViaCopyLink: 2.0,
		VideoQualityView: 1.5,
		Dwell:            1.0,
		DwellTime:        0.5,
		Click:            0.5,
		ProfileClick:     0.5,
		PhotoExpand:      0.3,
		QuotedClick:      0.3,
		FollowAuthor:     1.0,
		NotInterested:    -74.0,
		BlockAuthor:      -74.0,
		MuteAuthor:       -74.0,
		Report:           -369.0,
	}
}

// WeightsSum returns the sum of all positive action weights.
func (w WeightSet) WeightsSum() float64 {
	sum := 0.0
	for _, v := range w.positiveValues() {
		sum += v
	}
	return sum
}

// NegativeWeightsSum returns the absolute sum of all negative action weights.
func (w WeightSet) NegativeWeightsSum() float64 {
	sum := 0.0
	for _, v := range w.negativeValues() {
		if v < 0 {
			sum += -v
		}
	}
	return sum
}

func (w WeightSet) positiveValues() []float64 {
	return []float64{
		w.Favorite, w.Reply, w.Retweet, w.Quote, w.Share, w.ShareViaDM, w.ShareViaCopyLink,
		w.VideoQualityView, w.Dwell, w.DwellTime, w.Click, w.ProfileClick,
		w.PhotoExpand, w.QuotedClick, w.FollowAuthor,
	}
}

func (w WeightSet) negativeValues() []float64 {
	return []float64{w.NotInterested, w.BlockAuthor, w.MuteAuthor, w.Report}
}
