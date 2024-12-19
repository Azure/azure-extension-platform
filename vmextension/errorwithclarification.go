// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package vmextension

type ErrorWithClarification struct {
	ErrorCode int
	Err       error
}

func (ewc ErrorWithClarification) Error() string {
	return ewc.Err.Error()
}

func NewErrorWithClarification(errorCode int, err error) ErrorWithClarification {
	return ErrorWithClarification{
		ErrorCode: errorCode,
		Err:       err,
	}
}
