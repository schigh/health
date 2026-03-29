package health

import "errors"

// ErrHealth is the sentinel error for all health check errors.
// Use [errors.Is] to check if an error originated from this package.
var ErrHealth = errors.New("health")
