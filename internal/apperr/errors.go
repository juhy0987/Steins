// Package apperr defines the application-level error type and codes used
// across the API and service layers of Steins.
//
// apperr 패키지는 Steins API와 service 레이어에서 사용하는 구조화 에러 타입과
// 에러 코드 상수를 정의합니다.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Category classifies an error for logging, retry, and HTTP status mapping.
type Category string

const (
	CategoryValidation Category = "validation"
	CategoryAuth       Category = "auth"
	CategoryForbidden  Category = "forbidden"
	CategoryNotFound   Category = "not_found"
	CategoryConflict   Category = "conflict"
	CategoryRateLimit  Category = "rate_limit"
	CategoryNetwork    Category = "network"
	CategoryTimeout    Category = "timeout"
	CategoryDatabase   Category = "database"
	CategoryStorage    Category = "storage"
	CategoryInternal   Category = "internal"
)

// Error codes for v0. Domain-specific codes use UPPER_SNAKE names.
const (
	CodeValMissing      = "VAL_001"
	CodeValBadFormat    = "VAL_002"
	CodeValBadJSON      = "VAL_005"
	CodeMangaNotFound   = "MANGA_NOT_FOUND"
	CodeChapterNotFound = "CHAPTER_NOT_FOUND"
	CodePageNotFound    = "PAGE_NOT_FOUND"
	CodeImageNotFound   = "IMAGE_NOT_FOUND"
	CodeStorageRead     = "STORAGE_001"
	CodeStorageWrite    = "STORAGE_002"
	CodeInternal        = "INTERNAL_ERROR"
)

// Error is the structured application error type.
//
// Error는 application 전반에서 사용하는 구조화 에러 타입입니다.
type Error struct {
	Category   Category
	Code       string
	Message    string
	Resource   string
	ResourceID string
	StatusCode int
	Details    map[string]any
	Err        error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Category, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Category, e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code || e.Category == t.Category
}

// HTTPStatus returns the HTTP status code mapped from the category.
func (e *Error) HTTPStatus() int {
	if e.StatusCode != 0 {
		return e.StatusCode
	}
	switch e.Category {
	case CategoryValidation:
		return http.StatusBadRequest
	case CategoryAuth:
		return http.StatusUnauthorized
	case CategoryForbidden:
		return http.StatusForbidden
	case CategoryNotFound:
		return http.StatusNotFound
	case CategoryConflict:
		return http.StatusConflict
	case CategoryRateLimit:
		return http.StatusTooManyRequests
	case CategoryTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// NewValidationError constructs a validation error with the given code.
func NewValidationError(code, message string, err error) *Error {
	return &Error{
		Category: CategoryValidation,
		Code:     code,
		Message:  message,
		Err:      err,
	}
}

// NewNotFoundError constructs a 404 error for the given resource type and id.
func NewNotFoundError(code, resource, id string) *Error {
	return &Error{
		Category:   CategoryNotFound,
		Code:       code,
		Message:    resource + " not found",
		Resource:   resource,
		ResourceID: id,
	}
}

// NewStorageError constructs an internal storage error.
func NewStorageError(code, message string, err error) *Error {
	return &Error{
		Category: CategoryStorage,
		Code:     code,
		Message:  message,
		Err:      err,
	}
}

// NewInternalError constructs a generic internal error.
func NewInternalError(message string, err error) *Error {
	return &Error{
		Category: CategoryInternal,
		Code:     CodeInternal,
		Message:  message,
		Err:      err,
	}
}

// As is a thin wrapper around errors.As for *Error.
func As(err error) (*Error, bool) {
	var ae *Error
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
