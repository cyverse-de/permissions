package resource_types

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/cyverse-de/permissions/models"
)

// PutResourceTypesIDOKCode is the HTTP code returned for type PutResourceTypesIDOK
const PutResourceTypesIDOKCode int = 200

/*PutResourceTypesIDOK Updated

swagger:response putResourceTypesIdOK
*/
type PutResourceTypesIDOK struct {

	/*
	  In: Body
	*/
	Payload *models.ResourceTypeOut `json:"body,omitempty"`
}

// NewPutResourceTypesIDOK creates PutResourceTypesIDOK with default headers values
func NewPutResourceTypesIDOK() *PutResourceTypesIDOK {
	return &PutResourceTypesIDOK{}
}

// WithPayload adds the payload to the put resource types Id o k response
func (o *PutResourceTypesIDOK) WithPayload(payload *models.ResourceTypeOut) *PutResourceTypesIDOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the put resource types Id o k response
func (o *PutResourceTypesIDOK) SetPayload(payload *models.ResourceTypeOut) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PutResourceTypesIDOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// PutResourceTypesIDBadRequestCode is the HTTP code returned for type PutResourceTypesIDBadRequest
const PutResourceTypesIDBadRequestCode int = 400

/*PutResourceTypesIDBadRequest Bad Request

swagger:response putResourceTypesIdBadRequest
*/
type PutResourceTypesIDBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorOut `json:"body,omitempty"`
}

// NewPutResourceTypesIDBadRequest creates PutResourceTypesIDBadRequest with default headers values
func NewPutResourceTypesIDBadRequest() *PutResourceTypesIDBadRequest {
	return &PutResourceTypesIDBadRequest{}
}

// WithPayload adds the payload to the put resource types Id bad request response
func (o *PutResourceTypesIDBadRequest) WithPayload(payload *models.ErrorOut) *PutResourceTypesIDBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the put resource types Id bad request response
func (o *PutResourceTypesIDBadRequest) SetPayload(payload *models.ErrorOut) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PutResourceTypesIDBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// PutResourceTypesIDNotFoundCode is the HTTP code returned for type PutResourceTypesIDNotFound
const PutResourceTypesIDNotFoundCode int = 404

/*PutResourceTypesIDNotFound Not Found

swagger:response putResourceTypesIdNotFound
*/
type PutResourceTypesIDNotFound struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorOut `json:"body,omitempty"`
}

// NewPutResourceTypesIDNotFound creates PutResourceTypesIDNotFound with default headers values
func NewPutResourceTypesIDNotFound() *PutResourceTypesIDNotFound {
	return &PutResourceTypesIDNotFound{}
}

// WithPayload adds the payload to the put resource types Id not found response
func (o *PutResourceTypesIDNotFound) WithPayload(payload *models.ErrorOut) *PutResourceTypesIDNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the put resource types Id not found response
func (o *PutResourceTypesIDNotFound) SetPayload(payload *models.ErrorOut) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PutResourceTypesIDNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// PutResourceTypesIDInternalServerErrorCode is the HTTP code returned for type PutResourceTypesIDInternalServerError
const PutResourceTypesIDInternalServerErrorCode int = 500

/*PutResourceTypesIDInternalServerError Internal Server Error

swagger:response putResourceTypesIdInternalServerError
*/
type PutResourceTypesIDInternalServerError struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorOut `json:"body,omitempty"`
}

// NewPutResourceTypesIDInternalServerError creates PutResourceTypesIDInternalServerError with default headers values
func NewPutResourceTypesIDInternalServerError() *PutResourceTypesIDInternalServerError {
	return &PutResourceTypesIDInternalServerError{}
}

// WithPayload adds the payload to the put resource types Id internal server error response
func (o *PutResourceTypesIDInternalServerError) WithPayload(payload *models.ErrorOut) *PutResourceTypesIDInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the put resource types Id internal server error response
func (o *PutResourceTypesIDInternalServerError) SetPayload(payload *models.ErrorOut) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PutResourceTypesIDInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(500)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}