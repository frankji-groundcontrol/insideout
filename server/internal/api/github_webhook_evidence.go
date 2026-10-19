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
	} else if event == "deployment" || event == "deployment_status" {
		return planDeployment(event, ev, repoName)
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

// planDeployment maps a deployment or deployment_status delivery (rv4-FEAT-07).
// GitHub fires `deployment` when a deployment is created, before anyone knows
// whether it worked, so creation is plain activity. PRODUCT.md promises a
// deployment "may show release", so `release` is recorded only when a
// deployment_status reports success. A failure appends its own activity row
// ("failed deployments append evidence; they do not erase history"), and the
// (node, kind, detail) idempotency key makes GitHub's redeliveries free. Any
// other state (queued, pending, in_progress, inactive) refines nothing.
func planDeployment(event string, ev *webhookEvent, repoName string) (guideEvidence, bool) {
	what, kind := "deploy started", "activity"
	if event == "deployment_status" {
		switch ev.DeploymentStatus.State {
		case "success":
			what, kind = "deploy succeeded", "release"
		case "failure", "error":
			what = "deploy failed"
		default:
			return guideEvidence{}, false
		}
	}
	env := ev.Deployment.Environment
	if env == "" {
		env = "unknown environment"
	}
	p := guideEvidence{kind: kind, branch: strings.TrimPrefix(ev.Deployment.Ref, "refs/heads/"), detail: what + ": " + env}
	if sha := ev.Deployment.SHA; sha != "" {
		short := sha
		if len(short) > 7 {
			short = short[:7]
		}
		p.detail += " @ " + short
		p.source = fmt.Sprintf("https://github.com/%s/commit/%s", repoName, sha)
	}
	return p, true
}

// firstLine is the subject of a commit message or PR title: up to the first
// newline, at most 120 characters. The cap counts characters, not bytes, so it
// never cuts a multibyte one in two (R1-12): PostgreSQL rejects the invalid
// UTF-8 that leaves in the evidence detail.
func firstLine(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	if r := []rune(s); len(r) > 120 {
		return string(r[:120])
	}
	return s
}
