// Package validate provides pure Go validation utilities without external dependencies.
//
// All functions return plain error using fmt.Errorf("%s: %s", field, msg) format.
// Use JoinErrors to combine multiple validation errors into a single error.
package validate

import (
	"fmt"
	"strings"
)

// Err builds an error with format "field: msg".
func Err(field, msg string) error {
	return fmt.Errorf("%s: %s", field, msg)
}

// JoinErrors returns a single error concatenating messages with "; ", skipping nils.
// Returns nil if all errors are nil.
func JoinErrors(errs ...error) error {
	var nonNilErrs []string

	for _, err := range errs {
		if err != nil {
			nonNilErrs = append(nonNilErrs, err.Error())
		}
	}

	if len(nonNilErrs) == 0 {
		return nil
	}

	return fmt.Errorf("%s", strings.Join(nonNilErrs, "; "))
}
