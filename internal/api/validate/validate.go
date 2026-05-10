// Package validate provides request decoding and validation for the GForce API.
package validate

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	v        = validator.New()
	slugRe   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
)

func init() {
	// "slug": lowercase alphanumeric with interior hyphens, min length 1
	if err := v.RegisterValidation("slug", func(fl validator.FieldLevel) bool {
		return slugRe.MatchString(fl.Field().String())
	}); err != nil {
		panic("validate: registering slug: " + err.Error())
	}
}

// DecodeAndValidate decodes the JSON body of r into a T and validates it.
// Returns a human-readable error string on failure (suitable for API responses).
func DecodeAndValidate[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := Validate(v); err != nil {
		return v, err
	}
	return v, nil
}

// Validate validates a struct and returns a human-readable error string.
func Validate(val any) error {
	if err := v.Struct(val); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			return errors.New(formatErrors(verrs))
		}
		return err
	}
	return nil
}

func formatErrors(errs validator.ValidationErrors) string {
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		field := strings.ToLower(e.Field())
		switch e.Tag() {
		case "required":
			msgs = append(msgs, field+" is required")
		case "min":
			msgs = append(msgs, fmt.Sprintf("%s must be at least %s characters", field, e.Param()))
		case "max":
			msgs = append(msgs, fmt.Sprintf("%s must be at most %s characters", field, e.Param()))
		case "email":
			msgs = append(msgs, field+" must be a valid email address")
		case "url":
			msgs = append(msgs, field+" must be a valid URL")
		case "slug":
			msgs = append(msgs, field+" must be lowercase alphanumeric with hyphens (e.g. my-repo)")
		default:
			msgs = append(msgs, fmt.Sprintf("%s failed validation: %s", field, e.Tag()))
		}
	}
	return strings.Join(msgs, "; ")
}
