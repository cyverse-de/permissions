package resources

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-swagger/go-swagger/httpkit"

	"permissions/models"
)

/*DeleteResourceOK OK

swagger:response deleteResourceOK
*/
type DeleteResourceOK struct {
}

// NewDeleteResourceOK creates DeleteResourceOK with default headers values
func NewDeleteResourceOK() *DeleteResourceOK {
	return &DeleteResourceOK{}
}

// WriteResponse to the client
func (o *DeleteResourceOK) WriteResponse(rw http.ResponseWriter, producer httpkit.Producer) {

	rw.WriteHeader(200)
}

/*DeleteResourceNotFound Not Found

swagger:response deleteResourceNotFound
*/
type DeleteResourceNotFound struct {

	// In: body
	Payload *models.ErrorOut `json:"body,omitempty"`
}

// NewDeleteResourceNotFound creates DeleteResourceNotFound with default headers values
func NewDeleteResourceNotFound() *DeleteResourceNotFound {
	return &DeleteResourceNotFound{}
}

// WithPayload adds the payload to the delete resource not found response
func (o *DeleteResourceNotFound) WithPayload(payload *models.ErrorOut) *DeleteResourceNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete resource not found response
func (o *DeleteResourceNotFound) SetPayload(payload *models.ErrorOut) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteResourceNotFound) WriteResponse(rw http.ResponseWriter, producer httpkit.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		if err := producer.Produce(rw, o.Payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

/*DeleteResourceInternalServerError delete resource internal server error

swagger:response deleteResourceInternalServerError
*/
type DeleteResourceInternalServerError struct {

	// In: body
	Payload *models.ErrorOut `json:"body,omitempty"`
}

// NewDeleteResourceInternalServerError creates DeleteResourceInternalServerError with default headers values
func NewDeleteResourceInternalServerError() *DeleteResourceInternalServerError {
	return &DeleteResourceInternalServerError{}
}

// WithPayload adds the payload to the delete resource internal server error response
func (o *DeleteResourceInternalServerError) WithPayload(payload *models.ErrorOut) *DeleteResourceInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete resource internal server error response
func (o *DeleteResourceInternalServerError) SetPayload(payload *models.ErrorOut) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteResourceInternalServerError) WriteResponse(rw http.ResponseWriter, producer httpkit.Producer) {

	rw.WriteHeader(500)
	if o.Payload != nil {
		if err := producer.Produce(rw, o.Payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}