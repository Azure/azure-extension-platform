package vmextension

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/VMApplication-Extension/VmExtensionHelper/handlerenv"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// agentDir is where the agent is located, a subdirectory of which we use as the data directory
const agentDir = "/var/lib/waagent"

// most recent sequence, which was previously traced by seqNumFile. This was
// incorrect. The correct way is mrseq.  This file is auto-preserved by the agent.
const mostRecentSequence = "mrseq"

// HandlerEnvFileName is the file name of the Handler Environment as placed by the
// Azure Linux Guest Agent.
const handlerEnvFileName = "HandlerEnvironment.json"

// HandlerEnvironment describes the handler environment configuration presented
// to the extension handler by the Azure Linux Guest Agent.
type HandlerEnvironmentLinux struct {
	Version            float64 `json:"version"`
	Name               string  `json:"name"`
	HandlerEnvironment struct {
		HeartbeatFile string `json:"heartbeatFile"`
		StatusFolder  string `json:"statusFolder"`
		ConfigFolder  string `json:"configFolder"`
		LogFolder     string `json:"logFolder"`
	}
}

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

// GetHandlerEnv locates the HandlerEnvironment.json file by assuming it lives
// next to or one level above the extension handler (read: this) executable,
// reads, parses and returns it.
func getHandlerEnvironment(name string, version string) (he *handlerenv.HandlerEnvironment, _ error) {
	contents, _, err := findAndReadFile(handlerEnvFileName)
	if err != nil {
		return nil, err
	}

	handlerEnvLinux, err := parseHandlerEnv(contents)
	if err != nil {
		return nil, err
	}

	// The data directory is a subdirectory of waagent, with the extension name
	dataFolder := path.Join(agentDir, name)

	return &handlerenv.HandlerEnvironment{
		HeartbeatFile: handlerEnvLinux.HandlerEnvironment.HeartbeatFile,
		StatusFolder:  handlerEnvLinux.HandlerEnvironment.StatusFolder,
		ConfigFolder:  handlerEnvLinux.HandlerEnvironment.ConfigFolder,
		LogFolder:     handlerEnvLinux.HandlerEnvironment.LogFolder,
		DataFolder:    dataFolder,
	}, nil
}

// findAndReadFile locates the specified file on disk relative to our currently
// executing process and attempts to read the file
func findAndReadFile(fileName string) (b []byte, fileLoc string, _ error) {
	dir, err := scriptDir()
	if err != nil {
		return nil, "", fmt.Errorf("vmextension: cannot find base directory of the running process: %v", err)
	}

	paths := []string{
		filepath.Join(dir, fileName),       // this level (i.e. executable is in [EXT_NAME]/.)
		filepath.Join(dir, "..", fileName), // one up (i.e. executable is in [EXT_NAME]/bin/.)
	}

	for _, p := range paths {
		o, err := ioutil.ReadFile(p)
		if err != nil && !os.IsNotExist(err) {
			return nil, "", fmt.Errorf("vmextension: error examining '%s' at '%s': %v", fileName, p, err)
		} else if err == nil {
			fileLoc = p
			b = o
			break
		}
	}

	if b == nil {
		return nil, "", errNotFound
	}

	return b, fileLoc, nil
}

// scriptDir returns the absolute path of the running process.
func scriptDir() (string, error) {
	p, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "", err
	}
	return filepath.Dir(p), nil
}

// ParseHandlerEnv parses the
// /var/lib/waagent/[extension]/HandlerEnvironment.json format.
func parseHandlerEnv(b []byte) (*HandlerEnvironmentLinux, error) {
	var hf []HandlerEnvironmentLinux

	if err := json.Unmarshal(b, &hf); err != nil {
		return nil, fmt.Errorf("vmextension: failed to parse handler env: %v", err)
	}
	if len(hf) != 1 {
		return nil, fmt.Errorf("vmextension: expected 1 config in parsed HandlerEnvironment, found: %v", len(hf))
	}
	return &hf[0], nil
}
