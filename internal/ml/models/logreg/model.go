package logreg

import (
	"encoding/json"
	"errors"
	"math"
)

type TrainOptions struct {
	LearningRate float64
	Epochs       int
	L2           float64
}

type Artifact struct {
	FeatureNames []string  `json:"feature_names"`
	Weights      []float64 `json:"weights"`
	Bias         float64   `json:"bias"`
	Means        []float64 `json:"means"`
	Stds         []float64 `json:"stds"`
	L2           float64   `json:"l2"`
	LearningRate float64   `json:"learning_rate"`
	Epochs       int       `json:"epochs"`
}

type Model struct {
	artifact Artifact
}

func DefaultTrainOptions() TrainOptions {
	return TrainOptions{
		LearningRate: 0.05,
		Epochs:       600,
		L2:           0.0001,
	}
}

func Train(samples [][]float64, labels []float64, featureNames []string, opts TrainOptions) (*Model, error) {
	if len(samples) == 0 || len(samples) != len(labels) {
		return nil, errors.New("invalid training dataset")
	}
	if len(samples[0]) == 0 {
		return nil, errors.New("empty feature vectors")
	}
	if opts.LearningRate <= 0 {
		opts.LearningRate = DefaultTrainOptions().LearningRate
	}
	if opts.Epochs <= 0 {
		opts.Epochs = DefaultTrainOptions().Epochs
	}
	if opts.L2 < 0 {
		opts.L2 = DefaultTrainOptions().L2
	}

	featCount := len(samples[0])
	means := make([]float64, featCount)
	stds := make([]float64, featCount)
	for j := 0; j < featCount; j++ {
		for i := range samples {
			means[j] += samples[i][j]
		}
		means[j] /= float64(len(samples))
		for i := range samples {
			d := samples[i][j] - means[j]
			stds[j] += d * d
		}
		stds[j] = math.Sqrt(stds[j] / float64(len(samples)))
		if stds[j] == 0 {
			stds[j] = 1
		}
	}

	weights := make([]float64, featCount)
	bias := 0.0

	for epoch := 0; epoch < opts.Epochs; epoch++ {
		grads := make([]float64, featCount)
		gradBias := 0.0
		n := float64(len(samples))
		for i := range samples {
			x := normalize(samples[i], means, stds)
			p := sigmoid(dot(weights, x) + bias)
			err := p - labels[i]
			for j := range grads {
				grads[j] += err * x[j]
			}
			gradBias += err
		}
		for j := range weights {
			grads[j] = grads[j]/n + opts.L2*weights[j]
			weights[j] -= opts.LearningRate * grads[j]
		}
		bias -= opts.LearningRate * (gradBias / n)
	}

	if len(featureNames) != featCount {
		featureNames = defaultFeatureNames(featCount)
	}

	return &Model{artifact: Artifact{
		FeatureNames: featureNames,
		Weights:      weights,
		Bias:         bias,
		Means:        means,
		Stds:         stds,
		L2:           opts.L2,
		LearningRate: opts.LearningRate,
		Epochs:       opts.Epochs,
	}}, nil
}

func (m *Model) PredictProb(sample []float64) float64 {
	if m == nil || len(sample) != len(m.artifact.Weights) {
		return 0.5
	}
	x := normalize(sample, m.artifact.Means, m.artifact.Stds)
	return sigmoid(dot(m.artifact.Weights, x) + m.artifact.Bias)
}

func (m *Model) PredictBatch(samples [][]float64) []float64 {
	probs := make([]float64, len(samples))
	for i := range samples {
		probs[i] = m.PredictProb(samples[i])
	}
	return probs
}

func (m *Model) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, errors.New("nil model")
	}
	return json.Marshal(m.artifact)
}

func UnmarshalBinary(data []byte) (*Model, error) {
	if len(data) == 0 {
		return nil, errors.New("empty artifact")
	}
	var a Artifact
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	if len(a.Weights) == 0 || len(a.Weights) != len(a.Means) || len(a.Weights) != len(a.Stds) {
		return nil, errors.New("invalid artifact")
	}
	return &Model{artifact: a}, nil
}

func (m *Model) FeatureNames() []string {
	if m == nil {
		return nil
	}
	out := make([]string, len(m.artifact.FeatureNames))
	copy(out, m.artifact.FeatureNames)
	return out
}

func sigmoid(x float64) float64 {
	if x > 35 {
		return 1
	}
	if x < -35 {
		return 0
	}
	return 1 / (1 + math.Exp(-x))
}

func dot(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	s := 0.0
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

func normalize(in, means, stds []float64) []float64 {
	out := make([]float64, len(in))
	for i := range in {
		out[i] = (in[i] - means[i]) / stds[i]
	}
	return out
}

func defaultFeatureNames(n int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = "f" + formatInt(i)
	}
	return out
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
