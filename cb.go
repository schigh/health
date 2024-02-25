package health

import (
	"context"
)

// Runner is any function that is dispatched by a circuit breaker. It is intended
// to wrap a protected area of logic such that it is not called while a circuit
// breaker is open.
type Runner func(context.Context) (any, error)

// CircuitBreaker is a protective wrapper around any logic where an error
// tolerance can be exceeded, putting the circuit breaker into an open state.
// When a circuit breaker is open, the logic protected by the circuit breaker is
// not called.  When a circuit breaker is backing off, the protected logic is
// called according to the backoff strategy of the circuit breaker.
// Circuit breakers do not directly affect the health of the application, but are
// an indicator of the application's ability to perform its tasks.
type CircuitBreaker interface {
	Run(context.Context, Runner) (any, error)
}
