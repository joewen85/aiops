package errors

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e AppError) Error() string {
	return e.Message
}

var (
	ErrUnauthorized = AppError{Code: 1001, Message: "unauthorized"}
	ErrForbidden    = AppError{Code: 2001, Message: "forbidden"}
	ErrBadRequest   = AppError{Code: 3001, Message: "bad request"}
	ErrNotFound     = AppError{Code: 4004, Message: "not found"}
	ErrInternal     = AppError{Code: 5000, Message: "internal error"}
)

func New(code int, message string) AppError {
	return AppError{Code: code, Message: message}
}
