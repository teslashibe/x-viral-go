package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"
	viral "github.com/teslashibe/x-viral-go"
)

// GetWeightsInput is the typed input for xviral_get_weights. It is empty
// because the call takes no arguments.
type GetWeightsInput struct{}

func getWeights(_ context.Context, s *viral.Scorer, _ GetWeightsInput) (any, error) {
	w := s.Weights()
	return map[string]any{
		"weights":               w,
		"positive_weights_sum":  w.WeightsSum(),
		"negative_weights_sum":  w.NegativeWeightsSum(),
	}, nil
}

var weightsTools = []mcptool.Tool{
	mcptool.Define[*viral.Scorer, GetWeightsInput](
		"xviral_get_weights",
		"Return the active weighted-scorer action weights (and their positive/negative aggregates)",
		"Weights",
		getWeights,
	),
}
