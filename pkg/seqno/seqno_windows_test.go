// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package seqno

import (
	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
	"testing"
)

const (
	testKeyName          = "Software\\Microsoft\\Windows Azure\\HandlerState\\yaba_5.0"
	testExtensionName    = "yaba"
	testExtensionVersion = "5.0"
)

func Test_getSequenceNumberInternalNoRegistryKey(t *testing.T) {
	ensureRegistryKeyMissing(t, testKeyName)
	sn, err := getSequenceNumberInternal(testExtensionName, testExtensionVersion)
	require.Error(t, err, extensionerrors.ErrNotFound)
	require.Equal(t, uint(0), sn)
}

func Test_getSequenceNumberInternalNoValue(t *testing.T) {
	ensureRegistryKeyCreated(t, testKeyName)
	ensureRegistryValueMissing(t, testKeyName, sequenceNumberKeyName)
	sn, err := getSequenceNumberInternal(testExtensionName, testExtensionVersion)
	require.Error(t, err, extensionerrors.ErrNotFound)
	require.Equal(t, uint(0), sn)
}

func Test_getSequenceNumberInternalHasValue(t *testing.T) {
	ensureRegistryKeyCreated(t, testKeyName)
	ensureRegistryValueCreated(t, testKeyName, sequenceNumberKeyName, 5)
	sn, err := getSequenceNumberInternal(testExtensionName, testExtensionVersion)
	require.NoError(t, err, "getSequenceNumberInternal failed")
	require.Equal(t, uint(5), sn)
}

func Test_setSequenceNumberInternalNoRegistryKey(t *testing.T) {
	ensureRegistryKeyMissing(t, testKeyName)
	err := setSequenceNumberInternal(testExtensionName, testExtensionVersion, 42)
	require.Error(t, err, extensionerrors.ErrNotFound)
}

func Test_setSequenceNumberInternalValidReplace(t *testing.T) {
	ensureRegistryKeyCreated(t, testKeyName)
	ensureRegistryValueCreated(t, testKeyName, sequenceNumberKeyName, 5)
	err := setSequenceNumberInternal(testExtensionName, testExtensionVersion, 42)
	require.NoError(t, err, "setSequenceNumberInternal failed")
	sn, err := getSequenceNumberInternal(testExtensionName, testExtensionVersion)
	require.NoError(t, err, "getSquenceNumberInternal failed")
	require.Equal(t, uint(42), sn)
}

func Test_setSequenceNumberInternalValidSet(t *testing.T) {
	ensureRegistryKeyCreated(t, testKeyName)
	ensureRegistryValueMissing(t, testKeyName, sequenceNumberKeyName)
	err := setSequenceNumberInternal(testExtensionName, testExtensionVersion, 42)
	require.NoError(t, err, "setSequenceNumberInternal failed")
	sn, err := getSequenceNumberInternal(testExtensionName, testExtensionVersion)
	require.NoError(t, err, "getSquenceNumberInternal failed")
	require.Equal(t, uint(42), sn)
}

func ensureRegistryKeyCreated(t *testing.T, registryKey string) {
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, registryKey, registry.READ)
	defer k.Close()
	require.NoError(t, err, "Registry CreateKey failed")
}

func ensureRegistryKeyMissing(t *testing.T, registryKey string) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, registryKey, registry.READ)
	k.Close()
	if err != registry.ErrNotExist {
		err = registry.DeleteKey(registry.LOCAL_MACHINE, registryKey)
		require.NoError(t, err, "Registry DeleteKey failed")
	}
}

func ensureRegistryValueCreated(t *testing.T, registryKey string, valueName string, valueValue uint32) {
	ensureRegistryKeyCreated(t, registryKey)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, registryKey, registry.WRITE)
	defer k.Close()
	require.NoError(t, err, "Registry OpenKey failed")

	err = k.SetDWordValue(valueName, valueValue)
	require.NoError(t, err, "Registry SetDWordValue failed")
}

func ensureRegistryValueMissing(t *testing.T, registryKey string, valueName string) {
	ensureRegistryKeyCreated(t, registryKey)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, registryKey, registry.WRITE)
	defer k.Close()
	require.NoError(t, err, "Registry OpenKey failed")

	_, _, err = k.GetIntegerValue(valueName)
	if err == registry.ErrNotExist {
		err = k.DeleteValue(valueName)
		require.NoError(t, err, "Registry DeleteValue failed")
	}
}

