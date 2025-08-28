// internal/docker/types.go
package docker

type BuildOptions struct {
	Dockerfile  string      // default: "Dockerfile"
	ContextPath string      // default: "."
	BuildArgs   [][2]string // KEY,VALUE (deterministic)
	Labels      [][2]string // optional

	FullRefs []string // e.g. ["reg/org/app:deadbeef","reg/org/app:latest"]

	Target    string   // optional multi-stage target
	Pull      bool     // docker build --pull
	NoCache   bool     // docker build --no-cache
	Push      bool     // push after build
	DryRun    bool     // print only
}
