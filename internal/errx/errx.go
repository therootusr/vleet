package errx

import (
	"errors"
	"fmt"
)

// ErrNotImplemented is used by skeleton implementations to explicitly signal
// "the shape is here, the behavior will be implemented later".
var ErrNotImplemented = errors.New("not implemented")

// NotImplemented returns an error wrapped with ErrNotImplemented, scoped to a feature.
func NotImplemented(feature string) error {
	if feature == "" {
		return ErrNotImplemented
	}
	return fmt.Errorf("%s: %w", feature, ErrNotImplemented)
}
