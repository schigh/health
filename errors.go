package health

import "errors"

var ErrAddCheckAlreadyRunning = errors.New("health: cannot add a health check to a running health instance")
