package domain

import "time"

type Asset struct {
	Symbol string
	Name   string
}

type SignalDirection string

const (
	DirectionLong  SignalDirection = "long"
	DirectionShort SignalDirection = "short"
	DirectionHold  SignalDirection = "hold"
)

const (
	IndicatorRSI            = "rsi"
	IndicatorMACD           = "macd"
	IndicatorBollinger      = "bollinger"
	IndicatorVolumeZ        = "volume_zscore"
	IndicatorMLLogRegUp4H   = "ml_logreg_up4h"
	IndicatorMLXGBoostUp4H  = "ml_xgboost_up4h"
	IndicatorMLEnsembleUp4H = "ml_ensemble_up4h"
)

type Signal struct {
	ID        int64           `json:"id"`
	Symbol    string          `json:"symbol"`
	Interval  string          `json:"interval"`
	Indicator string          `json:"indicator"`
	Timestamp time.Time       `json:"timestamp"`
	Risk      RiskLevel       `json:"risk"`
	Direction SignalDirection `json:"direction"`
	Details   string          `json:"details,omitempty"`
	Image     *SignalImageRef `json:"image,omitempty"`
}

type SignalImageRef struct {
	ImageID   int64     `json:"image_id"`
	MimeType  string    `json:"mime_type"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	ExpiresAt time.Time `json:"expires_at"`
}

type SignalImageData struct {
	Ref   SignalImageRef
	Bytes []byte
}

type SignalFilter struct {
	Symbol    string
	Risk      *RiskLevel
	Indicator string
	Limit     int
}

type Recommendation struct {
	Signal Signal
	Text   string
}

type RiskLevel int

const (
	RiskLevel1 RiskLevel = 1
	RiskLevel2 RiskLevel = 2
	RiskLevel3 RiskLevel = 3
	RiskLevel4 RiskLevel = 4
	RiskLevel5 RiskLevel = 5
)

func (r RiskLevel) IsValid() bool {
	return r >= RiskLevel1 && r <= RiskLevel5
}

type ConversationMessage struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

type MLFeatureRow struct {
	Symbol        string
	Interval      string
	OpenTime      time.Time
	Ret1H         float64
	Ret4H         float64
	Ret12H        float64
	Ret24H        float64
	Volatility6H  float64
	Volatility24H float64
	VolumeZ24H    float64
	RSI14         float64
	MACDLine      float64
	MACDSignal    float64
	MACDHist      float64
	BBPos         float64
	BBWidth       float64
	TargetUp4H    *bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type MLModelVersion struct {
	ID                 int64
	ModelKey           string
	Version            int
	FeatureSpecVersion string
	TrainedFrom        time.Time
	TrainedTo          time.Time
	TrainedAt          time.Time
	HyperparamsJSON    string
	MetricsJSON        string
	ArtifactFormat     string
	ArtifactBlob       []byte
	IsActive           bool
	ActivatedAt        *time.Time
	CreatedAt          time.Time
}

type MLPrediction struct {
	ID             int64
	Symbol         string
	Interval       string
	OpenTime       time.Time
	TargetTime     time.Time
	ModelKey       string
	ModelVersion   int
	ProbUp         float64
	Confidence     float64
	Direction      SignalDirection
	Risk           RiskLevel
	SignalID       *int64
	DetailsJSON    string
	CreatedAt      time.Time
	ResolvedAt     *time.Time
	ActualUp       *bool
	IsCorrect      *bool
	RealizedReturn *float64
}
