package serrors

import (
	"serrors"
	"fmt"

	httpstatus "github.com/go-kratos/kratos/v2/transport/http/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

const (
	// UnknownCode is unknown code for error info.
	UnknownCode = 500
	// UnknownReason is unknown reason for error info.
	UnknownReason = ""
	// SupportPackageIsVersion1 this constant should not be referenced by any other code.
	SupportPackageIsVersion1 = true
)

// Error is a status error.
type Error struct {
	Status
	cause error
}

func (e *Error) Error() string {
	return fmt.Sprintf("errcode: code = %d errmsg = %s result = %v cause = %v", e.Errcode, e.Errmsg, e.Result)
}

// Unwrap provides compatibility for Go 1.13 error chains.
func (e *Error) Unwrap() error { return e.cause }

// Is matches each error in the chain with the target value.
func (e *Error) Is(err error) bool {
	if se := new(Error); errors.As(err, &se) {
		return se.Errcode == e.Errcode
	}
	return false
}

// WithCause with the underlying cause of the error.
func (e *Error) WithCause(cause error) *Error {
	err := Clone(e)
	err.cause = cause
	return err
}

// WithMetadata with an MD formed by the mapping of key, value.
func (e *Error) WithMetadata(md map[string]string) *Error {
	err := Clone(e)
	err.Result = md
	return err
}

// GRPCStatus returns the Status represented by se.
func (e *Error) GRPCStatus() *status.Status {
	s, _ := status.New(httpstatus.ToGRPCCode(int(e.Errcode)), e.Errmsg).
		WithDetails(&errdetails.ErrorInfo{
			Metadata: e.Result,
		})
	return s
}

// New returns an error object for the code, message.
func New(code int, message string) *Error {
	return &Error{
		Status: Status{
			Errcode: int32(code),
			Errmsg:  message,
		},
	}
}

// Newf New(code fmt.Sprintf(format, a...))
func Newf(code int, format string, a ...interface{}) *Error {
	return New(code, fmt.Sprintf(format, a...))
}

// Errorf returns an error object for the code, message and error info.
func Errorf(code int, format string, a ...interface{}) error {
	return New(code, fmt.Sprintf(format, a...))
}

// Code returns the http code for an error.
// It supports wrapped errors.
func Code(err error) int {
	if err == nil {
		return 200 //nolint:gomnd
	}
	return int(FromError(err).Errcode)
}

// Clone deep clone error to a new error.
func Clone(err *Error) *Error {
	if err == nil {
		return nil
	}
	metadata := make(map[string]string, len(err.Result))
	for k, v := range err.Result {
		metadata[k] = v
	}
	return &Error{
		cause: err.cause,
		Status: Status{
			Errcode: err.Errcode,
			Errmsg:  err.Errmsg,
			Result:  metadata,
		},
	}
}

// FromError try to convert an error to *Error.
// It supports wrapped errors.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	if se := new(Error); errors.As(err, &se) {
		return se
	}
	gs, ok := status.FromError(err)
	if !ok {
		return New(UnknownCode, err.Error())
	}
	ret := New(
		httpstatus.FromGRPCCode(gs.Code()),
		gs.Message(),
	)
	for _, detail := range gs.Details() {
		switch d := detail.(type) {
		case *errdetails.ErrorInfo:
			return ret.WithMetadata(d.Metadata)
		}
	}
	return ret
}
