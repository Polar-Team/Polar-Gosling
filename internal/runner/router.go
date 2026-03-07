package runner

// TagRouter determines whether a job can be routed to a runner based on tags.
type TagRouter struct {
	RunnerTags []string
}

// CanRoute returns true if every tag required by the job is present in the runner's tag set.
// An empty jobTags slice (untagged job) is always routable.
func (r *TagRouter) CanRoute(jobTags []string) bool {
	runnerTagSet := make(map[string]struct{}, len(r.RunnerTags))
	for _, t := range r.RunnerTags {
		runnerTagSet[t] = struct{}{}
	}
	for _, required := range jobTags {
		if _, ok := runnerTagSet[required]; !ok {
			return false
		}
	}
	return true
}
