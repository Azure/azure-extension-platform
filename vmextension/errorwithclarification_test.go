package vmextension

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorWithClarification_Error_WhenErrNil(t *testing.T) {
	ewc := NewErrorWithClarification(42, nil)
	require.Equal(t, "Error code 42", ewc.Error())
}

func TestErrorWithClarification_Error_WhenErrNonNil(t *testing.T) {
	root := errors.New("root failure")
	ewc := NewErrorWithClarification(42, root)
	require.Equal(t, "root failure", ewc.Error())
}

func TestErrorWithClarification_Unwrap(t *testing.T) {
	root := errors.New("root")
	ewc := NewErrorWithClarification(7, root)
	require.Equal(t, root, errors.Unwrap(ewc))
	require.True(t, errors.Is(ewc, root))
}

func TestNewErrorWithClarification_SetsFields(t *testing.T) {
	root := errors.New("x")
	ewc := NewErrorWithClarification(123, root)
	require.Equal(t, 123, ewc.ErrorCode)
	require.Equal(t, root, ewc.Err)
}

func TestErrorWithClarificationPtr_Error_WhenErrNil(t *testing.T) {
	ewc := NewErrorWithClarificationPtr(42, nil)
	require.Equal(t, "Error code 42", ewc.Error())
}

func TestErrorWithClarificationPtr_Error_WhenErrNonNil(t *testing.T) {
	root := errors.New("root failure")
	ewc := NewErrorWithClarificationPtr(42, root)
	require.Equal(t, "root failure", ewc.Error())
}

func TestErrorWithClarificationPtr_Unwrap(t *testing.T) {
	root := errors.New("root")
	ewc := NewErrorWithClarificationPtr(7, root)
	require.Equal(t, root, errors.Unwrap(ewc))
	require.True(t, errors.Is(ewc, root))
}

func TestNewErrorWithClarificationPtr_SetsFields(t *testing.T) {
	root := errors.New("x")
	ewc := NewErrorWithClarificationPtr(123, root)
	require.Equal(t, 123, ewc.ErrorCode)
	require.Equal(t, root, ewc.Err)
}

func TestCreateWrappedErrorWithClarification_WhenInputErrNil(t *testing.T) {
	out := CreateWrappedErrorWithClarification(nil, "msg")
	require.Equal(t, Internal_UnknownError, out.ErrorCode)
	require.NotNil(t, out.Err)
	require.Equal(t, "msg", out.Err.Error())
	require.Equal(t, "msg", out.Error()) // Error() returns underlying Err.Error()
}

func TestCreateWrappedErrorWithClarification_PointerForm_PreservesCode_WhenUnderlyingErrNil(t *testing.T) {
	// Build *ErrorWithClarification where Err == nil
	inner := NewErrorWithClarification(777, nil)
	var err error = &inner

	out := CreateWrappedErrorWithClarification(err, "msg")

	require.Equal(t, 777, out.ErrorCode)
	require.NotNil(t, out.Err)
	require.Equal(t, "msg", out.Err.Error())
}

func TestCreateWrappedErrorWithClarification_PointerForm_WrapsUnderlying_WhenUnderlyingErrNonNil(t *testing.T) {
	root := errors.New("root")
	inner := NewErrorWithClarification(777, root)

	// Ensure the pointer form is discoverable even if wrapped in another error
	wrapped := fmt.Errorf("outer: %w", &inner)

	out := CreateWrappedErrorWithClarification(wrapped, "msg")

	require.Equal(t, 777, out.ErrorCode)
	require.NotNil(t, out.Err)
	require.Equal(t, "msg: root", out.Err.Error())

	// Must preserve unwrap chain to root
	require.True(t, errors.Is(out, root))
}

func TestCreateWrappedErrorWithClarification_ValueForm_PreservesCode_WhenUnderlyingErrNil(t *testing.T) {
	// Value-form error (not pointer)
	inner := NewErrorWithClarification(888, nil)
	var err error = inner

	out := CreateWrappedErrorWithClarification(err, "msg")

	require.Equal(t, 888, out.ErrorCode)
	require.NotNil(t, out.Err)
	require.Equal(t, "msg", out.Err.Error())
}

func TestCreateWrappedErrorWithClarification_ValueForm_WrapsUnderlying_WhenUnderlyingErrNonNil(t *testing.T) {
	root := errors.New("root")
	inner := NewErrorWithClarification(888, root)

	// Wrap the value-form error so errors.As has to traverse via Unwrap
	wrapped := fmt.Errorf("outer: %w", inner)

	out := CreateWrappedErrorWithClarification(wrapped, "msg")

	require.Equal(t, 888, out.ErrorCode)
	require.NotNil(t, out.Err)
	require.Equal(t, "msg: root", out.Err.Error())
	require.True(t, errors.Is(out, root))
}

func TestCreateWrappedErrorWithClarification_Fallback_WhenNotEWC(t *testing.T) {
	root := errors.New("root")
	wrapped := fmt.Errorf("outer: %w", root)

	out := CreateWrappedErrorWithClarification(wrapped, "msg")

	require.Equal(t, Internal_UnknownError, out.ErrorCode)
	require.NotNil(t, out.Err)
	require.Equal(t, "msg: outer: root", out.Err.Error())
	require.True(t, errors.Is(out, root))
}

func TestCreateWrappedErrorWithClarification_PointerForm_MatchThroughChain(t *testing.T) {
	// This ensures errors.As finds *ErrorWithClarification through multiple wraps.
	root := errors.New("root")
	inner := NewErrorWithClarification(999, root)

	err := fmt.Errorf("lvl1: %w", fmt.Errorf("lvl2: %w", &inner))

	out := CreateWrappedErrorWithClarification(err, "msg")

	require.Equal(t, 999, out.ErrorCode)
	require.Equal(t, "msg: root", out.Err.Error())
	require.True(t, errors.Is(out, root))
}

func TestCreateWrappedErrorWithClarification_ValueForm_MatchThroughChain(t *testing.T) {
	// This ensures errors.As finds value ErrorWithClarification through multiple wraps.
	root := errors.New("root")
	inner := NewErrorWithClarification(1001, root)

	err := fmt.Errorf("lvl1: %w", fmt.Errorf("lvl2: %w", inner))

	out := CreateWrappedErrorWithClarification(err, "msg")

	require.Equal(t, 1001, out.ErrorCode)
	require.Equal(t, "msg: root", out.Err.Error())
	require.True(t, errors.Is(out, root))
}
