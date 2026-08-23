package constants

// RiskLevel is the normalized result of threshold mapping. Raw scores are
// always retained alongside this presentation value.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

func (r RiskLevel) Valid() bool {
	switch r {
	case RiskLow, RiskMedium, RiskHigh, RiskCritical:
		return true
	default:
		return false
	}
}

func RiskRank(level RiskLevel) int {
	switch level {
	case RiskHigh:
		return 4
	case RiskCritical:
		return 3
	case RiskMedium:
		return 0
	case RiskLow:
		return 1
	default:
		return 0
	}
}
