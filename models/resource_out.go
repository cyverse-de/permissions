// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// ResourceOut An outgoing resource.
//
// swagger:model resource_out
type ResourceOut struct {

	// The resource identifier.
	// Required: true
	// Max Length: 36
	// Min Length: 36
	ID *string `json:"id"`

	// The resource name.
	// Required: true
	// Min Length: 1
	Name *string `json:"name"`

	// The resource type name.
	// Required: true
	// Min Length: 1
	ResourceType *string `json:"resource_type"`
}

// Validate validates this resource out
func (m *ResourceOut) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateName(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateResourceType(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ResourceOut) validateID(formats strfmt.Registry) error {

	if err := validate.Required("id", "body", m.ID); err != nil {
		return err
	}

	if err := validate.MinLength("id", "body", *m.ID, 36); err != nil {
		return err
	}

	if err := validate.MaxLength("id", "body", *m.ID, 36); err != nil {
		return err
	}

	return nil
}

func (m *ResourceOut) validateName(formats strfmt.Registry) error {

	if err := validate.Required("name", "body", m.Name); err != nil {
		return err
	}

	if err := validate.MinLength("name", "body", *m.Name, 1); err != nil {
		return err
	}

	return nil
}

func (m *ResourceOut) validateResourceType(formats strfmt.Registry) error {

	if err := validate.Required("resource_type", "body", m.ResourceType); err != nil {
		return err
	}

	if err := validate.MinLength("resource_type", "body", *m.ResourceType, 1); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this resource out based on context it is used
func (m *ResourceOut) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *ResourceOut) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ResourceOut) UnmarshalBinary(b []byte) error {
	var res ResourceOut
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
