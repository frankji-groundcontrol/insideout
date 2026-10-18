package apiclient

import (
	"net/http"
	"path"
	"time"
)

// How long one request may run, from the moment it is sent until its body has
// been read. They are variables so a test can shrink them to milliseconds.
//
// There is no timeout on the http.Client: one number for every route cut off the
// calls that wait on a model (REL-06, FULL-07) or let a hung server hold a
// caller for ever, and a route needs the deadline that fits what it does.
var (
	// quickDeadline is for a call the server answers from its database.
	quickDeadline = 20 * time.Second

	// aiDeadline is for a call whose handler waits on the planner's model. A
	// draft is one forced-tool call capped at 4096 output tokens, retried once
	// when the provider hiccups, with a 90 s idle watchdog on the stream (see
	// agent/taxonomy.go): on a slow provider it ends in minutes, not seconds.
	aiDeadline = 5 * time.Minute
)

// aiRoutes are the POST routes (path.Match patterns, relative to the base URL)
// whose handler drafts with the model. The routes that only write a tree the
// caller already holds (/build/confirm, /expand/confirm) are not among them, and
// neither is anything else: a new route that calls the model has to be added
// here, or it gets the quick deadline.
var aiRoutes = []string{
	"/prds/*/build",     // PlanMVP drafts a whole roadmap from a PRD
	"/roadmap/*/expand", // ExpandNode drafts a node's sub-tasks
}

// deadlineFor returns how long a method + relative path may run.
func deadlineFor(method, route string) time.Duration {
	if method == http.MethodPost {
		for _, pattern := range aiRoutes {
			if ok, _ := path.Match(pattern, route); ok {
				return aiDeadline
			}
		}
	}
	return quickDeadline
}
