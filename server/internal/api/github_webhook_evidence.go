package api

import (
	"fmt"
	"strings"
)

// webhookEvent is the subset of GitHub delivery payloads the evidence
// pipeline reads (push, pull_request, deployment, deployment_status).
type webhookEvent struct {
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	Ref     string `json:"ref"` // push: refs/heads/x
	Commits []struct {
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
		Removed  []string `json:"removed"`
		SHA      string   `json:"id"`
		Message  string   `json:"message"`
	} `json:"commits"`
	PullRequest struct {
		Head struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		HTMLURL  string `json:"html_url"`
		Title    string `json:"title"`
		Merged   bool   `json:"merged"`
		MergedAt string `json:"merged_at"`
	} `json:"pull_request"`
	Deployment struct {
		Ref         string `json:"ref"`
		Environment string `json:"environment"`
		SHA         string `json:"sha"`
	} `json:"deployment"`
	DeploymentStatus struct {
		State string `json:"state"`
	} `json:"deployment_status"`
}

// guideEvidence is what one delivery appends to matched leaf nodes: the
// evidence row fields plus the guide match inputs.
type guideEvidence struct {
	branch string
	kind   string
	detail string
	source string
	labels []string
	paths  []string
}

// planGuideEvidence maps one event payload to the evidence it appends.
// ok=false means the delivery appends nothing (e.g. a push that names no
// branch).
func planGuideEvidence(event string, ev *webhookEvent, repoName string) (guideEvidence, bool) {
	p := guideEvidence{kind: "activity"}
	p.branch = strings.TrimPrefix(ev.Ref, "refs/heads/")
	if event == "push" {
		if p.branch == "" {
			return guideEvidence{}, false
		}
		for _, c := range ev.Commits {
			p.paths = append(p.paths, c.Added...)
			p.paths = append(p.paths, c.Modified...)
			p.paths = append(p.paths, c.Removed...)
		}
		if len(p.paths) > 200 {
			p.paths = p.paths[:200]
		}
		if len(ev.Commits) > 0 {
			c := ev.Commits[len(ev.Commits)-1]
			sha := c.SHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			p.detail = fmt.Sprintf("push %s: %s", sha, firstLine(c.Message))
			p.source = fmt.Sprintf("https://github.com/%s/commit/%s", repoName, c.SHA)
		} else {
			p.detail = "push to " + p.branch
		}
	} else if event == "deployment" {
		// CC-01 (review 2): PRODUCT.md and the generated guide both promise
		// "a deployment may show release" — map it into the same evidence
		// pipeline with the release kind. GitHub fires `deployment` at
		// creation; success/failure refinement belongs to a later
		// deployment_status slice.
		p.branch = strings.TrimPrefix(ev.Deployment.Ref, "refs/heads/")
		p.kind = "release"
		env := ev.Deployment.Environment
		if env == "" {
			env = "unknown environment"
		}
		if ev.Deployment.SHA != "" {
			sha := ev.Deployment.SHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			p.detail = fmt.Sprintf("deploy %s @ %s", env, sha)
			p.source = fmt.Sprintf("https://github.com/%s/commit/%s", repoName, ev.Deployment.SHA)
		} else {
			p.detail = fmt.Sprintf("deploy %s", env)
		}
	} else if event == "deployment_status" {
		// FEAT-07 (the deferred CC-01 slice): GitHub reports a deployment's
		// outcome only in this later delivery. A failure appends its own
		// activity row — PRODUCT.md: "failed deployments append evidence;
		// they do not erase history" — so the release row from creation is
		// never rewritten, and (node, kind, detail) dedupe makes GitHub's
		// redeliveries free. Any other state refines nothing.
		if state := ev.DeploymentStatus.State; state != "failure" && state != "error" {
			return guideEvidence{}, false
		}
		p.branch = strings.TrimPrefix(ev.Deployment.Ref, "refs/heads/")
		env := ev.Deployment.Environment
		if env == "" {
			env = "unknown environment"
		}
		if ev.Deployment.SHA != "" {
			sha := ev.Deployment.SHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			p.detail = fmt.Sprintf("deploy failed: %s @ %s", env, sha)
			p.source = fmt.Sprintf("https://github.com/%s/commit/%s", repoName, ev.Deployment.SHA)
		} else {
			p.detail = fmt.Sprintf("deploy failed: %s", env)
		}
	} else { // pull_request
		p.branch = ev.PullRequest.Head.Ref
		for _, l := range ev.PullRequest.Labels {
			p.labels = append(p.labels, l.Name)
		}
		switch {
		case ev.PullRequest.Merged:
			p.kind = "implementation"
		default:
			p.kind = "review"
		}
		p.detail = fmt.Sprintf("PR %s", firstLine(ev.PullRequest.Title))
		p.source = ev.PullRequest.HTMLURL
	}
	return p, true
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	if len(s) > 120 {
		return s[:120]
	}
	return s
}
