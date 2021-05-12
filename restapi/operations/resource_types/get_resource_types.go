// Code generated by go-swagger; DO NOT EDIT.

package resource_types

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// GetResourceTypesHandlerFunc turns a function with the right signature into a get resource types handler
type GetResourceTypesHandlerFunc func(GetResourceTypesParams) middleware.Responder

// Handle executing the request and returning a response
func (fn GetResourceTypesHandlerFunc) Handle(params GetResourceTypesParams) middleware.Responder {
	return fn(params)
}

// GetResourceTypesHandler interface for that can handle valid get resource types params
type GetResourceTypesHandler interface {
	Handle(GetResourceTypesParams) middleware.Responder
}

// NewGetResourceTypes creates a new http.Handler for the get resource types operation
func NewGetResourceTypes(ctx *middleware.Context, handler GetResourceTypesHandler) *GetResourceTypes {
	return &GetResourceTypes{Context: ctx, Handler: handler}
}

/* GetResourceTypes swagger:route GET /resource_types resource_types getResourceTypes

List Resource Types

Lists resource types known to the permissions service. A resource type represents a class of resources to which permissions may be applied. For example, the Discovery environment has apps collectively defined as a single resource type in the permissions service.

*/
type GetResourceTypes struct {
	Context *middleware.Context
	Handler GetResourceTypesHandler
}

func (o *GetResourceTypes) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewGetResourceTypesParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
