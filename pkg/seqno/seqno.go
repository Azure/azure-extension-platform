package seqno

import (
	"fmt"
	"github.com/D1v38om83r/azure-extension-platform/pkg/extensionerrors"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
)

type ISequenceNumberRetriever interface {
	GetSequenceNumber(name, version string) (uint, error)
}

type ProcSequenceNumberRetriever struct {
}

func (*ProcSequenceNumberRetriever) GetSequenceNumber(name, version string) (uint, error) {
	return getSequenceNumberInternal(name, version)
}

// GetCurrentSequenceNumber returns the current sequence number the extension is using
func GetCurrentSequenceNumber(ctx log.Logger, retriever ISequenceNumberRetriever, name, version string) (sn uint, _ error) {
	sequenceNumber, err := retriever.GetSequenceNumber(name, version)
	if err == extensionerrors.ErrNotFound {
		// If we can't find the sequence number, then it's possible that the extension
		// hasn't been installed yet. Go back to 0.
		ctx.Log("message", "couldn't find sequence number")
		return 0, nil
	}

	return sequenceNumber, err
}

func SetSequenceNumber(extName, extVersion string, seqNo uint) error {
	return setSequenceNumberInternal(extName, extVersion, seqNo)
}

// findSeqnum finds the most reecently used file under the config folder
// Note that this is different than just choosing the highest number, which may be incorrect
func FindSeqNum(ctx log.Logger, configFolder string) (uint, error) {
	g, err := filepath.Glob(configFolder + "/*.settings")
	if err != nil {
		return 0, err
	}

	// Start by finding the file with the latest time
	files, err := ioutil.ReadDir(configFolder)
	if err != nil {
		return 0, err
	}

	var modTime time.Time
	var names []string
	for _, fi := range files {
		if fi.Mode().IsRegular() && strings.HasSuffix(fi.Name(), ".settings") {
			if !fi.ModTime().Before(modTime) {
				if fi.ModTime().After(modTime) {
					modTime = fi.ModTime()
					names = names[:0]
				}

				// Unlikely - but just in case the files have the same time
				names = append(names, fi.Name())
			}
		}
	}

	if len(names) == 0 {
		ctx.Log("message", "findSeqNum failed", "error", fmt.Errorf("Cannot find the seqNo from %s. Not enough files", configFolder))
		return 0, extensionerrors.ErrNoSettingsFiles
	} else if len(names) == 1 {
		i, err := strconv.Atoi(strings.Replace(names[0], ".settings", "", 1))
		if err != nil {
			ctx.Log("message", "findSeqNum failed", "error", fmt.Errorf("Can't parse int from filename: %s", names[0]))
			return 0, extensionerrors.ErrInvalidSettingsFileName
		}

		return uint(i), nil
	} else {
		// For some reason we have two or more files with the same time stamp.
		// Revert to choosing the highest number.
		seqs := make([]int, len(g))
		for _, v := range g {
			f := filepath.Base(v)
			i, err := strconv.Atoi(strings.Replace(f, ".settings", "", 1))
			if err != nil {
				ctx.Log("message", "findSeqNum failed", "error", fmt.Errorf("Can't parse int from filename: %s", f))
				return 0, extensionerrors.ErrInvalidSettingsFileName
			}
			seqs = append(seqs, i)
		}

		sort.Sort(sort.Reverse(sort.IntSlice(seqs)))
		return uint(seqs[0]), nil
	}
}
