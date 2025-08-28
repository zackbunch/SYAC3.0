package runtime

type Flow string

const (
	FlowAuto    Flow = "auto"
	FlowFeature Flow = "feature"
	FlowMR      Flow = "mr"
	FlowDefault Flow = "default"
	FlowRelease Flow = "release"
)

func ResolveFlow(ctx Context, forced Flow) Flow {
	if forced != FlowAuto {
		return forced
	}
	switch {
	case ctx.IsTag:
		return FlowRelease
	case ctx.IsMergeRequest:
		return FlowMR
	case ctx.IsDefaultBranch:
		return FlowDefault
	case ctx.IsFeatureBranch:
		return FlowFeature
	default:
		return FlowDefault
	}
}
