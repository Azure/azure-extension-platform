// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package vmextension

import (
	"errors"
	"fmt"
)

const Internal_UnknownError = -9999

type ErrorWithClarification struct {
	ErrorCode int
	Err       error
}

func (ewc ErrorWithClarification) Error() string {
	if ewc.Err == nil {
		return fmt.Sprintf("Error code %d", ewc.ErrorCode)
	}

	return ewc.Err.Error()
}

func (ewc ErrorWithClarification) Unwrap() error { return ewc.Err }

func NewErrorWithClarification(errorCode int, err error) ErrorWithClarification {
	return ErrorWithClarification{
		ErrorCode: errorCode,
		Err:       err,
	}
}

func CreateWrappedErrorWithClarification(err error, msg string) ErrorWithClarification {
	if err == nil {
		return NewErrorWithClarification(Internal_UnknownError, errors.New(msg))
	}

	// Try Pointer form
	var ewc *ErrorWithClarification
	if errors.As(err, &ewc) && ewc != nil {
		// Preserve existing ErrorCode, replace/wrap underlying Err.
		if ewc.Err == nil {
			return NewErrorWithClarification(ewc.ErrorCode, errors.New(msg))
		}
		return NewErrorWithClarification(ewc.ErrorCode, fmt.Errorf("%s: %w", msg, ewc.Err))
	}

	// Try value form
	var ewcVal ErrorWithClarification
	if errors.As(err, &ewcVal) {
		if ewcVal.Err == nil {
			return NewErrorWithClarification(ewcVal.ErrorCode, errors.New(msg))
		}
		return NewErrorWithClarification(ewcVal.ErrorCode, fmt.Errorf("%s: %w", msg, ewcVal.Err))
	}

	return NewErrorWithClarification(Internal_UnknownError, fmt.Errorf("%s: %w", msg, err))
}
