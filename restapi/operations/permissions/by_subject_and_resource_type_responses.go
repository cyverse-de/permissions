package permissions

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/cyverse-de/permissions/models"
)

// BySubjectAndResourceTypeOKCode is the HTTP code returned for type BySubjectAndResourceTypeOK
const BySubjectAndResourceTypeOKCode int = 200

/*BySubjectAndResourceTypeOK OK

swagger:response bySubjectAndResourceTypeOK
*/
type BySubjectAndResourceTypeOK struct {

	/*
	  In: Body
	*/
	Payload *models.PermissionList `json:"body,omitempty"`
}

// NewBySubjectAndResourceTypeOK creates BySubjectAndResourceTypeOK with default headers values
func NewBySubjectAndResourceTypeOK() *BySubjectAndResourceTypeOK {
	return &BySubjectAndResourceTypeOK{}
}

// WithPayload adds the payload to the by subject and resource type o k response
func (o *BySubjectAndResourceTypeOK) WithPayload(payload *models.PermissionList) *BySubjectAndResourceTypeOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the by subject and resource type o k response
func (o *BySubjectAndResourceTypeOK) SetPayload(payload *models.PermissionList) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *BySubjectAndResourceTypeOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// BySubjectAndResourceTypeBadRequestCode is the HTTP code returned for type BySubjectAndResourceTypeBadRequest
const BySubjectAndResourceTypeBadRequestCode int = 400

/*BySubjectAndResourceTypeBadRequest Bad Request

swagger:response bySubjectAndResourceTypeBadRequest
*/
type BySubjectAndResourceTypeBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorOut `json:"body,omitempty"`
}

// NewBySubjectAndResourceTypeBadRequest creates BySubjectAndResourceTypeBadRequest with default headers values
func NewBySubjectAndResourceTypeBadRequest() *BySubjectAndResourceTypeBadRequest {
	return &BySubjectAndResourceTypeBadRequest{}
}

// WithPayload adds the payload to the by subject and resource type bad request response
func (o *BySubjectAndResourceTypeBadRequest) WithPayload(payload *models.ErrorOut) *BySubjectAndResourceTypeBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the by subject and resource type bad request response
func (o *BySubjectAndResourceTypeBadRequest) SetPayload(payload *models.ErrorOut) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *BySubjectAndResourceTypeBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// BySubjectAndResourceTypeInternalServerErrorCode is the HTTP code returned for type BySubjectAndResourceTypeInternalServerError
const BySubjectAndResourceTypeInternalServerErrorCode int = 500

/*BySubjectAndResourceTypeInternalServerError Internal Server Error

swagger:response bySubjectAndResourceTypeInternalServerError
*/
type BySubjectAndResourceTypeInternalServerError struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorOut `json:"body,omitempty"`
}

// NewBySubjectAndResourceTypeInternalServerError creates BySubjectAndResourceTypeInternalServerError with default headers values
func NewBySubjectAndResourceTypeInternalServerError() *BySubjectAndResourceTypeInternalServerError {
	return &BySubjectAndResourceTypeInternalServerError{}
}

// WithPayload adds the payload to the by subject and resource type internal server error response
func (o *BySubjectAndResourceTypeInternalServerError) WithPayload(payload *models.ErrorOut) *BySubjectAndResourceTypeInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the by subject and resource type internal server error response
func (o *BySubjectAndResourceTypeInternalServerError) SetPayload(payload *models.ErrorOut) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *BySubjectAndResourceTypeInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(500)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}