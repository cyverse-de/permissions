// Code generated by go-swagger; DO NOT EDIT.

package resource_types

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/cyverse-de/permissions/models"
)

// DeleteResourceTypesIDOKCode is the HTTP code returned for type DeleteResourceTypesIDOK
const DeleteResourceTypesIDOKCode int = 200

/*DeleteResourceTypesIDOK Deleted

swagger:response deleteResourceTypesIdOK
*/
type DeleteResourceTypesIDOK struct {
}

// NewDeleteResourceTypesIDOK creates DeleteResourceTypesIDOK with default headers values
func NewDeleteResourceTypesIDOK() *DeleteResourceTypesIDOK {

	return &DeleteResourceTypesIDOK{}
}

// WriteResponse to the client
func (o *DeleteResourceTypesIDOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(200)
}

// DeleteResourceTypesIDBadRequestCode is the HTTP code returned for type DeleteResourceTypesIDBadRequest
const DeleteResourceTypesIDBadRequestCode int = 400

/*DeleteResourceTypesIDBadRequest Bad Request

swagger:response deleteResourceTypesIdBadRequest
*/
type DeleteResourceTypesIDBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorOut `json:"body,omitempty"`
}

// NewDeleteResourceTypesIDBadRequest creates DeleteResourceTypesIDBadRequest with default headers values
func NewDeleteResourceTypesIDBadRequest() *DeleteResourceTypesIDBadRequest {

	return &DeleteResourceTypesIDBadRequest{}
}

// WithPayload adds the payload to the delete resource types Id bad request response
func (o *DeleteResourceTypesIDBadRequest) WithPayload(payload *models.ErrorOut) *DeleteResourceTypesIDBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete resource types Id bad request response
func (o *DeleteResourceTypesIDBadRequest) SetPayload(payload *models.ErrorOut) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteResourceTypesIDBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// DeleteResourceTypesIDNotFoundCode is the HTTP code returned for type DeleteResourceTypesIDNotFound
const DeleteResourceTypesIDNotFoundCode int = 404

/*DeleteResourceTypesIDNotFound Not Found

swagger:response deleteResourceTypesIdNotFound
*/
type DeleteResourceTypesIDNotFound struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorOut `json:"body,omitempty"`
}

// NewDeleteResourceTypesIDNotFound creates DeleteResourceTypesIDNotFound with default headers values
func NewDeleteResourceTypesIDNotFound() *DeleteResourceTypesIDNotFound {

	return &DeleteResourceTypesIDNotFound{}
}

// WithPayload adds the payload to the delete resource types Id not found response
func (o *DeleteResourceTypesIDNotFound) WithPayload(payload *models.ErrorOut) *DeleteResourceTypesIDNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete resource types Id not found response
func (o *DeleteResourceTypesIDNotFound) SetPayload(payload *models.ErrorOut) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteResourceTypesIDNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// DeleteResourceTypesIDInternalServerErrorCode is the HTTP code returned for type DeleteResourceTypesIDInternalServerError
const DeleteResourceTypesIDInternalServerErrorCode int = 500

/*DeleteResourceTypesIDInternalServerError Internal Server Error

swagger:response deleteResourceTypesIdInternalServerError
*/
type DeleteResourceTypesIDInternalServerError struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorOut `json:"body,omitempty"`
}

// NewDeleteResourceTypesIDInternalServerError creates DeleteResourceTypesIDInternalServerError with default headers values
func NewDeleteResourceTypesIDInternalServerError() *DeleteResourceTypesIDInternalServerError {

	return &DeleteResourceTypesIDInternalServerError{}
}

// WithPayload adds the payload to the delete resource types Id internal server error response
func (o *DeleteResourceTypesIDInternalServerError) WithPayload(payload *models.ErrorOut) *DeleteResourceTypesIDInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete resource types Id internal server error response
func (o *DeleteResourceTypesIDInternalServerError) SetPayload(payload *models.ErrorOut) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteResourceTypesIDInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(500)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
