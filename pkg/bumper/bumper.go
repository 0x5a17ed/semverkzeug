package bumper

import (
	"github.com/Masterminds/semver/v3"
)

type partFunc func(semver.Version) semver.Version

func (f partFunc) bump(inp semver.Version) semver.Version {
	return f(inp)
}

type Part interface {
	bump(inp semver.Version) semver.Version
}

var (
	Major Part = partFunc(semver.Version.IncMajor)
	Minor Part = partFunc(semver.Version.IncMinor)
	Patch Part = partFunc(semver.Version.IncPatch)
)

// Bump calculates a new semantic version by incrementing the
// specified part of the provided version.
func Bump(ov semver.Version, part Part) semver.Version {
	return part.bump(ov)
}
