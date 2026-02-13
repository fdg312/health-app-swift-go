package emailotp

import "fmt"

// ServiceError is a typed API-level error returned by OTP service.
type ServiceError struct {
	Status  int
	Code    string
	Message string
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func AsServiceError(err error) (*ServiceError, bool) {
	typed, ok := err.(*ServiceError)
	return typed, ok
}
