package version

import (
	"fmt"
	"log"
	"runtime"
	"time"
)

var (
	version      = ""                     // Injected with a linker flag
	buildDate    = "1970-01-01T00:00:00Z" // Injected with a linker flag
	gitCommit    = ""                     // Injected with a linker flag
	gitTreeState = ""                     // Injected with a linker flag
)

// Version encapsulates all available information about the source code and the
// build.
type Version struct {
	// Version is a human-friendly version string.
	Version string `json:"version"`
	// BuildDate is the date/time on which the application was built.
	BuildDate time.Time `json:"buildDate"`
	// GitCommit is the ID (sha) of the last commit to the application's source
	// code that is included in this build.
	GitCommit string `json:"gitCommit"`
	// GitTreeDirty is true if the application's source code contained
	// uncommitted changes at the time it was built; otherwise it is false.
	GitTreeDirty bool `json:"gitTreeDirty"`
	// GoVersion is the version of Go that was used to build the application.
	GoVersion string `json:"goVersion"`
	// Compiler indicates what Go compiler was used for the build.
	Compiler string `json:"compiler"`
	// Platform indicates the OS and CPU architecture for which the application
	// was built.
	Platform string `json:"platform"`
}

var ver Version

func init() {
	buildDate, err := time.Parse(time.RFC3339, buildDate)
	if err != nil {
		log.Fatal(err)
	}

	ver = Version{
		Version:      version,
		BuildDate:    buildDate,
		GitCommit:    gitCommit,
		GitTreeDirty: gitTreeState != "clean",
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	// If we're missing the version string or commit info, or if the tree is
	// dirty, dynamically formulate a version string from available info...
	if ver.Version == "" || ver.GitCommit == "" || ver.GitTreeDirty {
		// Override whatever version string we started with
		ver.Version = "devel"
		// Tack on commit info
		if len(ver.GitCommit) >= 7 {
			ver.Version = fmt.Sprintf("%s+%s", ver.Version, gitCommit[0:7])
		} else {
			ver.Version = fmt.Sprintf("%s+unknown", ver.Version)
		}
		// Indicate if the tree was dirty
		if ver.GitTreeDirty {
			ver.Version = fmt.Sprintf("%s.dirty", ver.Version)
		}
	}
}

func GetVersion() Version {
	return ver
}
