package vmextension

import (
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
)

// agentDir is where the agent is located, a subdirectory of which we use as the data directory
const agentDir = "/var/lib/waagent"

// most recent sequence, which was previously traced by seqNumFile. This was
// incorrect. The correct way is mrseq.  This file is auto-preserved by the agent.
const mostRecentSequence = "mrseq"

// GetOSName returns the name of the OS
func getOSName() (name string) {
	return "Linux"
}

// getSequenceNumberInternal is the Linux specific logic for reading the current
// sequence number for the extension
func getSequenceNumberInternal(name string, version string) (_ uint, _ error) {
	// Read the sequence number from the mrseq file
	b, _, err := findAndReadFile(mostRecentSequence)
	if err != nil {
		return 0, err
	}

	// TODO: add test for spaces when Linux unit tests are added
	contents := strings.TrimSpace(string(b))
	sequenceNumber64, err := strconv.ParseUint(contents, 10, 32)
	sequenceNumber := uint(sequenceNumber64)
	if err != nil {
		return 0, fmt.Errorf("vmextension: cannot read sequence number")
	}

	return sequenceNumber, nil
}

// setSequenceNumberInternal is the Linux specific logic for writing the sequence
// number to disk
func setSequenceNumberInternal(ve *VMExtension, seqNo uint) error {
	_, fileLoc, err := findAndReadFile(mostRecentSequence)
	if err != nil {
		return err
	}

	contents := string(seqNo)
	b := []byte(contents)
	return ioutil.WriteFile(fileLoc, b, 0644)
}

func getDataFolder(name string, version string) string {
	return path.Join(agentDir, name)
}
