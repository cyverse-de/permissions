package db

import (
	"github.com/cyverse-de/permissions/models"
)

// SubjectDTO is a data transfer object used for information about a subject.
type SubjectDTO struct {
	ID          *models.InternalSubjectID
	SubjectID   *models.ExternalSubjectID
	SubjectType *models.SubjectType
}

// ToSubjectOut converts a subject DTO to the output subject model.
func (s *SubjectDTO) ToSubjectOut() *models.SubjectOut {
	var subjectOut models.SubjectOut

	subjectOut.ID = s.ID
	subjectOut.SubjectID = s.SubjectID
	subjectOut.SubjectType = s.SubjectType
	subjectOut.SubjectSourceID = subjectSourceID(s.SubjectType)

	return &subjectOut
}
