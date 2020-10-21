package seqno

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os"
	"testing"
)

func Test_writeReadSequenceNumberFile(t *testing.T) {
	defer cleanupTest()
	seqno := uint(rand.Int())
	err := setSequenceNumberInternal("some name", "some version", seqno)
	assert.NoError(t, err, "set sequence number should succeed")
	readSeqno, err := getSequenceNumberInternal("some name", "some version")
	assert.NoError(t, err, "get sequence number should succeed")
	assert.Equal(t, seqno, readSeqno, "read sequence number should be same as set sequence number")
}

func cleanupTest() {
	os.Remove(mostRecentSequenceFileName)
}
