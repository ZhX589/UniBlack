package service

import "github.com/ZhX589/UniBlack/backend/internal/models"

// EventVisibleTo is the explicit read policy shared by direct reads, version
// history and events embedded in subject responses. Unknown statuses are not public.
func EventVisibleTo(event *models.Event, userID string, roles []string) bool {
	if event == nil {
		return false
	}
	if event.Status == "published" || event.Status == "corrected" {
		return true
	}
	for _, role := range roles {
		if role == "admin" || role == "moderator" {
			return true
		}
	}
	return userID != "" && event.SubmittedBy != nil && *event.SubmittedBy == userID
}

// HasGovernanceRole reports whether the role set carries a trusted governance
// marker (admin or moderator). Currently used by evidence write authorization.
func HasGovernanceRole(roles []string) bool {
	for _, role := range roles {
		if role == "admin" || role == "moderator" {
			return true
		}
	}
	return false
}

// canManageEvidence permits the original submitter or a governance role to
// modify evidence. An empty actor can never match an empty uploader.
func canManageEvidence(evidence *models.Evidence, actorID string, roles []string) bool {
	if actorID == "" || evidence == nil {
		return false
	}
	if HasGovernanceRole(roles) {
		return true
	}
	return evidence != nil && evidence.UploadedBy != nil && *evidence.UploadedBy == actorID
}
