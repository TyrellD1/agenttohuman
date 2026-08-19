package version

// Version is the release version (no leading v). Dev builds stay "dev".
var Version = "dev"

// Commit is the source revision, set at release time.
var Commit = ""

func String() string {
	if Version == "" {
		return "dev"
	}
	return Version
}
