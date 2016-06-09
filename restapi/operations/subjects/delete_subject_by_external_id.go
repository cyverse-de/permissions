package subjects

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	middleware "github.com/go-swagger/go-swagger/httpkit/middleware"
)

// DeleteSubjectByExternalIDHandlerFunc turns a function with the right signature into a delete subject by external Id handler
type DeleteSubjectByExternalIDHandlerFunc func(DeleteSubjectByExternalIDParams) middleware.Responder

// Handle executing the request and returning a response
func (fn DeleteSubjectByExternalIDHandlerFunc) Handle(params DeleteSubjectByExternalIDParams) middleware.Responder {
	return fn(params)
}

// DeleteSubjectByExternalIDHandler interface for that can handle valid delete subject by external Id params
type DeleteSubjectByExternalIDHandler interface {
	Handle(DeleteSubjectByExternalIDParams) middleware.Responder
}

// NewDeleteSubjectByExternalID creates a new http.Handler for the delete subject by external Id operation
func NewDeleteSubjectByExternalID(ctx *middleware.Context, handler DeleteSubjectByExternalIDHandler) *DeleteSubjectByExternalID {
	return &DeleteSubjectByExternalID{Context: ctx, Handler: handler}
}

/*DeleteSubjectByExternalID swagger:route DELETE /subjects subjects deleteSubjectByExternalId

Delete Subjects by External ID

Removes subjects (entities to which permissions may be gratned) from the database.

*/
type DeleteSubjectByExternalID struct {
	Context *middleware.Context
	Handler DeleteSubjectByExternalIDHandler
}

func (o *DeleteSubjectByExternalID) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, _ := o.Context.RouteInfo(r)
	var Params = NewDeleteSubjectByExternalIDParams()

	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request

	o.Context.Respond(rw, r, route.Produces, route, res)

}
