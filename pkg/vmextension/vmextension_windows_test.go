package vmextension

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_getOSName(t *testing.T) {
	osName := getOSName()
	require.Equal(t, "Windows", osName)
}
