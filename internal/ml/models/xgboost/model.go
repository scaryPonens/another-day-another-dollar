package xgboost

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"math"

	"github.com/rmera/boo"
	"github.com/rmera/boo/utils"
)

type TrainOptions struct {
	Rounds       int
	LearningRate float64
	MaxDepth     int
}

type artifact struct {
	FeatureNames []string `json:"feature_names"`
	ModelText    string   `json:"model_text"`
}

type Model struct {
	featureNames []string
	boost        *boo.MultiClass
}

func DefaultTrainOptions() TrainOptions {
	return TrainOptions{
		Rounds:       40,
		LearningRate: 0.08,
		MaxDepth:     4,
	}
}

func Train(samples [][]float64, labels []float64, featureNames []string, opts TrainOptions) (*Model, error) {
	if len(samples) == 0 || len(samples) != len(labels) {
		return nil, errors.New("invalid training dataset")
	}
	if len(samples[0]) == 0 {
		return nil, errors.New("empty feature vectors")
	}
	classSet := make(map[int]struct{}, 2)
	intLabels := make([]int, len(labels))
	for i, v := range labels {
		label := 0
		if v >= 0.5 {
			label = 1
		}
		intLabels[i] = label
		classSet[label] = struct{}{}
	}
	if len(classSet) < 2 {
		return nil, errors.New("xgboost requires at least two classes")
	}
	if opts.Rounds <= 0 {
		opts.Rounds = DefaultTrainOptions().Rounds
	}
	if opts.LearningRate <= 0 {
		opts.LearningRate = DefaultTrainOptions().LearningRate
	}
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = DefaultTrainOptions().MaxDepth
	}
	if len(featureNames) != len(samples[0]) {
		featureNames = make([]string, len(samples[0]))
		for i := range featureNames {
			featureNames[i] = "f"
		}
	}

	o := boo.DefaultXOptions()
	o.Rounds = opts.Rounds
	o.LearningRate = opts.LearningRate
	o.MaxDepth = opts.MaxDepth
	o.Verbose = false
	o.EarlyStop = 0

	data := &utils.DataBunch{
		Data:   samples,
		Labels: intLabels,
		Keys:   featureNames,
	}
	model := boo.NewMultiClass(data, o)
	if model == nil {
		return nil, errors.New("failed to train xgboost model")
	}
	return &Model{featureNames: append([]string(nil), featureNames...), boost: model}, nil
}

func (m *Model) PredictProb(sample []float64) float64 {
	if m == nil || m.boost == nil {
		return 0.5
	}
	probs := m.boost.PredictSingle(sample)
	labels := m.boost.ClassLabels()
	for i := range labels {
		if labels[i] == 1 {
			return clamp01(probs[i])
		}
	}
	if len(probs) == 0 {
		return 0.5
	}
	return clamp01(probs[len(probs)-1])
}

func (m *Model) PredictBatch(samples [][]float64) []float64 {
	out := make([]float64, len(samples))
	for i := range samples {
		out[i] = m.PredictProb(samples[i])
	}
	return out
}

func (m *Model) MarshalBinary() ([]byte, error) {
	if m == nil || m.boost == nil {
		return nil, errors.New("nil model")
	}
	var buf bytes.Buffer
	if err := boo.JSONMultiClass(m.boost, "softmax", &buf); err != nil {
		return nil, err
	}
	return json.Marshal(artifact{
		FeatureNames: m.featureNames,
		ModelText:    buf.String(),
	})
}

func UnmarshalBinary(blob []byte) (*Model, error) {
	if len(blob) == 0 {
		return nil, errors.New("empty artifact")
	}
	var a artifact
	if err := json.Unmarshal(blob, &a); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(stringsToReader(a.ModelText))
	model, err := boo.UnJSONMultiClass(reader)
	if err != nil {
		return nil, err
	}
	return &Model{featureNames: append([]string(nil), a.FeatureNames...), boost: model}, nil
}

func (m *Model) FeatureNames() []string {
	if m == nil {
		return nil
	}
	out := make([]string, len(m.featureNames))
	copy(out, m.featureNames)
	return out
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) {
		return 0.5
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func stringsToReader(s string) *bytes.Reader {
	return bytes.NewReader([]byte(s))
}
