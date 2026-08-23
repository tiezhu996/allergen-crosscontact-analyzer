package constants

type AssessmentStatus string

const (
	AssessmentQueued        AssessmentStatus = "queued"
	AssessmentCalculating   AssessmentStatus = "calculating"
	AssessmentPendingReview AssessmentStatus = "pending_review"
	AssessmentAccepted      AssessmentStatus = "accepted"
	AssessmentRejected      AssessmentStatus = "rejected"
	AssessmentStale         AssessmentStatus = "stale"
)

func (s AssessmentStatus) Valid() bool {
	switch s {
	case AssessmentQueued, AssessmentCalculating, AssessmentPendingReview,
		AssessmentAccepted, AssessmentRejected, AssessmentStale:
		return true
	default:
		return false
	}
}

func CanTransition(from, to AssessmentStatus) bool {
	switch from {
	case AssessmentQueued:
		return to == AssessmentCalculating
	case AssessmentCalculating:
		return to == AssessmentPendingReview
	case AssessmentPendingReview:
		return to == AssessmentAccepted || to == AssessmentRejected || to == AssessmentStale
	case AssessmentAccepted, AssessmentRejected:
		return to == AssessmentStale
	default:
		return false
	}
}

type Role string

const (
	RoleQualityAnalyst Role = "quality_analyst"
	RoleReviewer       Role = "reviewer"
	RoleAdmin          Role = "admin"
)

func (r Role) Valid() bool {
	return r == RoleQualityAnalyst || r == RoleReviewer || r == RoleAdmin
}
