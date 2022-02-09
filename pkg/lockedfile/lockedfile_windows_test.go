// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package lockedfile

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSplittingAndCombiningUlongToUintHighLow(t *testing.T) {
	var longInt uint64 = 0xABCD1234FEDC

	high, low := splitUlongToTwoUint32(longInt)
	assert.Equal(t, uint32(0xABCD), high, "values should be equal")
	assert.Equal(t, uint32(0x1234FEDC), low, "values should be equal")

	result := combineTwoUint32ToUlong(high, low)
	assert.Equal(t, longInt, result, "values should be equal")

	longInt += uint64(0xF0000000)
	high, low = splitUlongToTwoUint32(longInt)
	assert.Equal(t, uint32(0xABCE), high, "values should be equal")
	assert.Equal(t, uint32(0x0234FEDC), low, "values should be equal")
}
