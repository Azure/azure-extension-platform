package vmextension

import (
	"testing"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
	"github.com/D1v38om83r/azure-extension-platform/pkg/extensionerrors"
)

func Test_initializationInfoValidate(t *testing.T) {
	// Empty name
	_, err := GetInitializationInfo("", "5.0", true, testEnableCallback)
	require.Equal(t, extensionerrors.ErrArgCannotBeNullOrEmpty, err)

	// Empty version
	_, err = GetInitializationInfo("yaba", "", true, testEnableCallback)
	require.Equal(t, extensionerrors.ErrArgCannotBeNullOrEmpty, err)

	// Null enable callback
	_, err = GetInitializationInfo("yaba", "5.0", true, nil)
	require.Equal(t, extensionerrors.ErrArgCannotBeNull, err)
}

func Test_initializationInfoDefaults(t *testing.T) {
	ii, err := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
	require.NoError(t, err, "Error from initialization")
	require.Equal(t, "yaba", ii.Name)
	require.Equal(t, "5.0", ii.Version)
	require.Equal(t, true, ii.SupportsDisable)
	require.Equal(t, true, ii.RequiresSeqNoChange)
	require.Equal(t, 52, ii.InstallExitCode)
	require.Equal(t, 3, ii.OtherExitCode)
}

func testEnableCallback(ctx log.Logger, ext *VMExtension) (string, error) {
	return "blah", nil
}
