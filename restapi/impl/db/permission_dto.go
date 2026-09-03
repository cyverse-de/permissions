package db

import (
	"github.com/cyverse-de/permissions/models"
)

// PermissionDTO is a data transfer object for permissions.
type PermissionDTO struct {
	ID                *models.PermissionID
	InternalSubjectID *models.InternalSubjectID
	SubjectID         *models.ExternalSubjectID
	SubjectType       *models.SubjectType
	ResourceID        string
	ResourceName      string
	ResourceType      string
	PermissionLevel   *models.PermissionLevel
}

// Subject source IDs, as Grouper reported them. Callers -- terrain among them --
// tell users and groups apart by this field, so the values remain a wire
// contract even though Grouper no longer supplies them.
const (
	userSourceID  = models.SubjectSourceID("ldap")
	groupSourceID = models.SubjectSourceID("g:gsa")
)

// subjectSourceID derives a subject's source from its type.
func subjectSourceID(subjectType *models.SubjectType) *models.SubjectSourceID {
	source := userSourceID
	if subjectType != nil && *subjectType == models.SubjectTypeGroup {
		source = groupSourceID
	}
	return &source
}

// ToPermission converts a permission data transfer object to a permission object.
func (p *PermissionDTO) ToPermission() *models.Permission {

	// Extract the subject.
	subject := &models.SubjectOut{
		ID:              p.InternalSubjectID,
		SubjectID:       p.SubjectID,
		SubjectType:     p.SubjectType,
		SubjectSourceID: subjectSourceID(p.SubjectType),
	}

	// Extract the resource.
	resource := &models.ResourceOut{
		ID:           &p.ResourceID,
		Name:         &p.ResourceName,
		ResourceType: &p.ResourceType,
	}

	// Extract the permission itself.
	permission := &models.Permission{
		ID:              p.ID,
		PermissionLevel: p.PermissionLevel,
		Resource:        resource,
		Subject:         subject,
	}

	return permission
}
