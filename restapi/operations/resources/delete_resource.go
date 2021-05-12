// Code generated by go-swagger; DO NOT EDIT.

package resources

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// DeleteResourceHandlerFunc turns a function with the right signature into a delete resource handler
type DeleteResourceHandlerFunc func(DeleteResourceParams) middleware.Responder

// Handle executing the request and returning a response
func (fn DeleteResourceHandlerFunc) Handle(params DeleteResourceParams) middleware.Responder {
	return fn(params)
}

// DeleteResourceHandler interface for that can handle valid delete resource params
type DeleteResourceHandler interface {
	Handle(DeleteResourceParams) middleware.Responder
}

// NewDeleteResource creates a new http.Handler for the delete resource operation
func NewDeleteResource(ctx *middleware.Context, handler DeleteResourceHandler) *DeleteResource {
	return &DeleteResource{Context: ctx, Handler: handler}
}

/* DeleteResource swagger:route DELETE /resources/{id} resources deleteResource

Delete a Resource

Removes a resource from the database.

*/
type DeleteResource struct {
	Context *middleware.Context
	Handler DeleteResourceHandler
}

func (o *DeleteResource) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewDeleteResourceParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
