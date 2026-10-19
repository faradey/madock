package snapshotengine

import "sync"

// ArchiveCoder gives a clone made from a tar snapshot its code another way than
// unpacking the snapshot's files.tar.gz into it. It answers whether it did; on
// false madock unpacks as before.
//
// Unpacking is a full copy of the code for every clone — 26 s and 3.8 GB for a
// Magento checkout, measured 2026-10-18 — and on a server whose filesystem has
// no copy-on-write it is the one cost an engine cannot take away.
type ArchiveCoder func(src Project, snapshotDir string, clone Project) (bool, error)

var archiveCoder ArchiveCoder

// SetArchiveCoder installs the ArchiveCoder.
//
// Extension point for madock-pro: there the code of a clone from a tar snapshot
// is an overlay over the snapshot's code, unpacked once, so ten clones cost what
// they write. Nothing here installs one.
func SetArchiveCoder(f ArchiveCoder) { archiveCoder = f }

// CodeFromArchive asks the ArchiveCoder, when there is one.
func CodeFromArchive(src Project, snapshotDir string, clone Project) (bool, error) {
	if archiveCoder == nil {
		return false, nil
	}
	return archiveCoder(src, snapshotDir, clone)
}

// InUseCheck says why a snapshot directory must not be removed, or "" when it
// may be.
type InUseCheck func(project Project, snapshotDir string) string

var (
	inUseMu     sync.RWMutex
	inUseChecks []InUseCheck
)

// RegisterInUse adds a check that removing a snapshot consults first.
//
// Extension point for madock-pro: a clone whose code is an overlay over a
// snapshot's unpacked code lives on that directory, and removing it would pull
// the code out from under a running project.
func RegisterInUse(f InUseCheck) {
	inUseMu.Lock()
	defer inUseMu.Unlock()
	inUseChecks = append(inUseChecks, f)
}

// InUse is the first reason a registered check gives, or "".
func InUse(project Project, snapshotDir string) string {
	inUseMu.RLock()
	defer inUseMu.RUnlock()
	for _, f := range inUseChecks {
		if reason := f(project, snapshotDir); reason != "" {
			return reason
		}
	}
	return ""
}
