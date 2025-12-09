package errs

import (
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
)

type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "VALIDATION_ERROR"
	ErrorTypeInternal      ErrorType = "INTERNAL_ERROR"
	ErrorTypeBadRequest    ErrorType = "BAD_REQUEST"
	ErrorTypeNotFound      ErrorType = "NOT_FOUND"
	ErrorTypeUnauthorized  ErrorType = "UNAUTHORIZED"
	ErrorTypeAlreadyExists ErrorType = "ALREADY_EXISTS"
)

type Error struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Err     error     `json:"-"`
}

func (e Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e Error) StatusCode() int {
	switch e.Type {
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeBadRequest:
		return http.StatusBadRequest
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeUnauthorized:
		return http.StatusUnauthorized
	case ErrorTypeAlreadyExists:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func (e Error) GrpcStatusCode() codes.Code {
	switch e.Type {
	case ErrorTypeValidation:
		return codes.InvalidArgument
	case ErrorTypeBadRequest:
		return codes.InvalidArgument
	case ErrorTypeNotFound:
		return codes.NotFound
	case ErrorTypeUnauthorized:
		return codes.Unauthenticated
	case ErrorTypeAlreadyExists:
		return codes.AlreadyExists
	default:
		return codes.Internal
	}
}

func New(errType ErrorType, message string, err error) Error {
	return Error{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

func ValidationError(message string, err error) Error {
	return New(ErrorTypeValidation, message, err)
}

func InternalError(message string, err error) Error {
	return New(ErrorTypeInternal, message, err)
}

func BadRequestError(message string, err error) Error {
	return New(ErrorTypeBadRequest, message, err)
}

func NotFoundError(item string, err error) Error {
	return New(ErrorTypeNotFound, fmt.Sprintf("%s not found", item), err)
}

func UnauthorizedError(message string, err error) Error {
	return New(ErrorTypeUnauthorized, message, err)
}

func AlreadyExistsError(item string, err error) Error {
	return New(ErrorTypeAlreadyExists, fmt.Sprintf("%s already exists", item), err)
}

func IsErrorType(err error, errType ErrorType) (bool, Error) {
	var e Error
	if errors.As(err, &e) && e.Type == errType {
		return true, e
	}
	return false, e
}

func ErrorCast(err error) Error {
	var e Error
	if errors.As(err, &e) {
		return e
	}
	return InternalError("internal server error", err)
}

