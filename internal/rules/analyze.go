package rules

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/oaslananka/cifence/internal/githubactions"
	"github.com/oaslananka/cifence/internal/parser"
	"gopkg.in/yaml.v3"
)

var fullSHA = regexp.MustCompile(`^[A-Fa-f0-9]{40}$`)

var mutableRefs = map[string]struct{}{
	"main":    {},
	"master":  {},
	"dev":     {},
	"develop": {},
	"trunk":   {},
	"latest":  {},
	"head":    {},
}

var unsafeCheckoutRefs = []string{
	"github.event.pull_request.head.ref",
	"github.head_ref",
	"github.event.pull_request.head.sha",
	"refs/pull/",
}

var untrustedContextRefs = []string{
	"github.head_ref",
	"github.event.inputs.",
	"inputs.",
	"github.event.pull_request.title",
	"github.event.pull_request.body",
	"github.event.pull_request.head.ref",
	"github.event.pull_request.head.label",
	"github.event.pull_request.head.repo.full_name",
	"github.event.issue.title",
	"github.event.issue.body",
	"github.event.comment.body",
	"github.event.head_commit.message",
	"github.event.commits.",
	"github.event.sender.login",
	"github.event.label.name",
	"github.event.release.name",
	"github.event.release.body",
	"github.event.discussion.title",
	"github.event.discussion.body",
}

var pullRequestTargetShellPatterns = []string{
	"gh pr checkout",
	"git fetch origin pull/",
	"github.event.pull_request.head.sha",
	"github.event.pull_request.head.ref",
}

var knownPermissionScopes = map[string]struct{}{
	"actions":              {},
	"artifact-metadata":    {},
	"attestations":         {},
	"checks":               {},
	"contents":             {},
	"deployments":          {},
	"discussions":          {},
	"id-token":             {},
	"issues":               {},
	"metadata":             {},
	"models":               {},
	"packages":             {},
	"pages":                {},
	"pull-requests":        {},
	"security-events":      {},
	"statuses":             {},
	"vulnerability-alerts": {},
}

var dangerousWriteScopes = map[string]struct{}{
	"actions":           {},
	"artifact-metadata": {},
	"attestations":      {},
	"checks":            {},
	"contents":          {},
	"deployments":       {},
	"discussions":       {},
	"id-token":          {},
	"issues":            {},
	"packages":          {},
	"pages":             {},
	"pull-requests":     {},
	"security-events":   {},
	"statuses":          {},
}

func Analyze(doc parser.Document) []githubactions.Finding {
	root := documentMapping(doc.Root)
	if root == nil {
		return nil
	}

	findings := make([]githubactions.Finding, 0)
	findings = append(findings, permissionsFindings(doc.File, root)...)
	findings = append(findings, actionReferenceFindings(doc.File, root)...)
	findings = append(findings, reusableWorkflowFindings(doc.File, root)...)
	findings = append(findings, injectionFindings(doc.File, root)...)
	findings = append(findings, containerImageFindings(doc.File, root)...)
	findings = append(findings, pullRequestTargetFindings(doc.File, root)...)
	findings = append(findings, workflowRunBoundaryFindings(doc.File, root)...)
	findings = append(findings, selfHostedRunnerFindings(doc.File, root)...)

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		if findings[i].Column != findings[j].Column {
			return findings[i].Column < findings[j].Column
		}
		return findings[i].RuleID < findings[j].RuleID
	})
	return findings
}

func ParseFinding(file string, err error) githubactions.Finding {
	definition := DefinitionByID(ParseRuleID)
	return githubactions.Finding{
		RuleID:      definition.ID,
		Severity:    definition.Severity,
		Title:       definition.Title,
		Message:     "Workflow YAML could not be parsed.",
		File:        file,
		Line:        1,
		Column:      1,
		Evidence:    sanitizeEvidence(err.Error()),
		Remediation: definition.Remediation,
	}
}

func permissionsFindings(file string, root *yaml.Node) []githubactions.Finding {
	var findings []githubactions.Finding
	permissionsKey, permissionsValue, hasWorkflowPermissions := lookup(root, "permissions")
	if hasWorkflowPermissions && isWriteAll(permissionsValue) {
		findings = append(findings, newFinding("CF-PERM-001", file, permissionsValue, "Workflow uses permissions: write-all.", "permissions: write-all", "permissions"))
	}
	if !hasWorkflowPermissions {
		findings = append(findings, newFinding("CF-PERM-002", file, root, "Workflow is missing an explicit permissions block.", "workflow permissions missing"))
	} else if permissionsKey != nil && isNullNode(permissionsValue) {
		findings = append(findings, newFinding("CF-PERM-002", file, permissionsKey, "Workflow permissions block is empty.", "permissions block empty"))
	}

	workflowPermissions := permissionSet(permissionsValue)
	if hasWorkflowPermissions {
		findings = append(findings, unknownPermissionFindings(file, permissionsValue, workflowPermissions, "workflow", "permissions")...)
	}
	prLike := hasEvent(root, "pull_request") || hasPullRequestTarget(root)
	if prLike {
		findings = append(findings, dangerousPermissionFindings(file, permissionsValue, workflowPermissions, "workflow", "CF-PERM-003", "permissions")...)
	}
	if hasIDTokenWrite(workflowPermissions) && !hasTrustedPushBranch(root) {
		findings = append(findings, newFinding("CF-PERM-004", file, permissionsValue, "Workflow grants id-token: write without a clear trusted branch restriction.", "id-token: write", "permissions.id-token"))
	}

	_, jobsNode, ok := lookup(root, "jobs")
	if !ok {
		return findings
	}
	for _, job := range mappingPairs(asMapping(jobsNode)) {
		jobMap := asMapping(job.Value)
		if jobMap == nil {
			continue
		}
		_, jobPermissionsValue, hasJobPermissions := lookup(jobMap, "permissions")
		if hasJobPermissions && isWriteAll(jobPermissionsValue) {
			message := fmt.Sprintf("Job %q uses permissions: write-all.", job.Key.Value)
			findings = append(findings, newFinding("CF-PERM-001", file, jobPermissionsValue, message, "permissions: write-all", fmt.Sprintf("jobs.%s.permissions", job.Key.Value)))
		}
		if hasJobPermissions && isNullNode(jobPermissionsValue) {
			message := fmt.Sprintf("Job %q permissions block is empty.", job.Key.Value)
			evidence := fmt.Sprintf("job %q permissions empty", job.Key.Value)
			findings = append(findings, newFinding("CF-PERM-002", file, job.Key, message, evidence))
		}
		if !hasJobPermissions {
			message := fmt.Sprintf("Job %q is missing an explicit permissions block.", job.Key.Value)
			evidence := fmt.Sprintf("job %q permissions missing", job.Key.Value)
			findings = append(findings, newFinding("CF-PERM-002", file, job.Key, message, evidence))
		}
		jobPermissions := permissionSet(jobPermissionsValue)
		if hasJobPermissions {
			findings = append(findings, unknownPermissionFindings(file, jobPermissionsValue, jobPermissions, fmt.Sprintf("job %q", job.Key.Value), fmt.Sprintf("jobs.%s.permissions", job.Key.Value))...)
		}
		if prLike {
			findings = append(findings, dangerousPermissionFindings(file, jobPermissionsValue, jobPermissions, fmt.Sprintf("job %q", job.Key.Value), "CF-PERM-003", fmt.Sprintf("jobs.%s.permissions", job.Key.Value))...)
		}
		if hasIDTokenWrite(jobPermissions) && !hasJobEnvironment(jobMap) && !hasTrustedPushBranch(root) {
			message := fmt.Sprintf("Job %q grants id-token: write without an environment or trusted branch restriction.", job.Key.Value)
			findings = append(findings, newFinding("CF-PERM-004", file, jobPermissionsValue, message, "id-token: write", fmt.Sprintf("jobs.%s.permissions.id-token", job.Key.Value)))
		}
		if hasJobPermissions && escalatesPermissions(workflowPermissions, jobPermissions) {
			message := fmt.Sprintf("Job %q grants write permissions beyond the workflow baseline.", job.Key.Value)
			findings = append(findings, newFinding("CF-PERM-005", file, jobPermissionsValue, message, fmt.Sprintf("job %q permission escalation", job.Key.Value), fmt.Sprintf("jobs.%s.permissions", job.Key.Value)))
		}
		effectivePermissions := workflowPermissions
		if hasJobPermissions {
			effectivePermissions = jobPermissions
		}
		if hasDangerousWrite(effectivePermissions) && jobHasThirdPartyAction(jobMap) {
			message := fmt.Sprintf("Job %q combines write permissions with a third-party action.", job.Key.Value)
			findings = append(findings, newFinding("CF-PERM-006", file, job.Key, message, fmt.Sprintf("job %q third-party action with write token", job.Key.Value), fmt.Sprintf("jobs.%s", job.Key.Value)))
		}
	}
	return findings
}

func actionReferenceFindings(file string, root *yaml.Node) []githubactions.Finding {
	var findings []githubactions.Finding
	for _, step := range steps(root) {
		_, usesNode, ok := lookup(step, "uses")
		if !ok {
			continue
		}
		usesValue, ok := scalarString(usesNode)
		if !ok || isLocalAction(usesValue) {
			continue
		}
		if strings.HasPrefix(strings.ToLower(usesValue), "docker://") {
			if dockerUsesLatest(usesValue) {
				findings = append(findings, newFinding("CF-ACT-002", file, usesNode, "Docker action uses the mutable latest tag.", usesValue))
			} else if !dockerUsesDigest(usesValue) {
				findings = append(findings, newFinding("CF-ACT-001", file, usesNode, "Docker action is not pinned to an immutable digest.", usesValue))
			}
			continue
		}
		action, ref, hasRef := splitActionRef(usesValue)
		if !hasRef || action == "" {
			findings = append(findings, newFinding("CF-ACT-001", file, usesNode, "Remote action reference is missing a full commit SHA.", usesValue))
			continue
		}
		if fullSHA.MatchString(ref) {
			continue
		}
		if isMutableRef(ref) {
			findings = append(findings, newFinding("CF-ACT-002", file, usesNode, "Remote action uses a mutable ref.", usesValue))
			continue
		}
		findings = append(findings, newFinding("CF-ACT-001", file, usesNode, "Remote action reference is not pinned to a full commit SHA.", usesValue))
	}
	return findings
}

func reusableWorkflowFindings(file string, root *yaml.Node) []githubactions.Finding {
	var findings []githubactions.Finding
	for _, job := range jobs(root) {
		jobMap := asMapping(job.Value)
		if jobMap == nil {
			continue
		}
		_, usesNode, ok := lookup(jobMap, "uses")
		if !ok {
			continue
		}
		usesValue, ok := scalarString(usesNode)
		if !ok || isLocalAction(usesValue) || !strings.Contains(usesValue, ".github/workflows/") {
			continue
		}
		if _, secretsNode, ok := lookup(jobMap, "secrets"); ok && isScalarValue(secretsNode, "inherit") {
			findings = append(findings, newFinding("CF-SEC-001", file, secretsNode, "Reusable workflow inherits all caller secrets.", "secrets: inherit", fmt.Sprintf("jobs.%s.secrets", job.Key.Value)))
		}
		action, ref, hasRef := splitActionRef(usesValue)
		if !hasRef || action == "" {
			findings = append(findings, newFinding("CF-ACT-003", file, usesNode, "Reusable workflow reference is missing a full commit SHA.", usesValue, fmt.Sprintf("jobs.%s.uses", job.Key.Value)))
		} else if fullSHA.MatchString(ref) {
			continue
		} else if isMutableRef(ref) {
			findings = append(findings, newFinding("CF-ACT-004", file, usesNode, "Reusable workflow uses a mutable ref.", usesValue, fmt.Sprintf("jobs.%s.uses", job.Key.Value)))
		} else {
			findings = append(findings, newFinding("CF-ACT-003", file, usesNode, "Reusable workflow is not pinned to a full commit SHA.", usesValue, fmt.Sprintf("jobs.%s.uses", job.Key.Value)))
		}
	}
	return findings
}

func injectionFindings(file string, root *yaml.Node) []githubactions.Finding {
	var findings []githubactions.Finding
	for _, step := range steps(root) {
		if _, runNode, ok := lookup(step, "run"); ok {
			if runValue, ok := scalarString(runNode); ok && containsUntrustedContext(runValue) {
				findings = append(findings, newFinding("CF-INJ-001", file, runNode, "Untrusted GitHub context is interpolated into a run step.", firstUntrustedContext(runValue), "jobs.*.steps[].run"))
				if containsEnvironmentFileSink(runValue) {
					findings = append(findings, newFinding("CF-ENV-001", file, runNode, "Untrusted GitHub context is written to a GitHub environment file.", firstUntrustedContext(runValue), "jobs.*.steps[].run"))
				}
			}
		}
		if _, shellNode, ok := lookup(step, "shell"); ok {
			if shellValue, ok := scalarString(shellNode); ok && containsUntrustedContext(shellValue) {
				findings = append(findings, newFinding("CF-INJ-001", file, shellNode, "Untrusted GitHub context is interpolated into a shell selector.", firstUntrustedContext(shellValue), "jobs.*.steps[].shell"))
			}
		}
		usesValue := ""
		if _, usesNode, ok := lookup(step, "uses"); ok {
			usesValue, _ = scalarString(usesNode)
		}
		if isGitHubScriptAction(usesValue) {
			if _, withNode, ok := lookup(step, "with"); ok {
				if _, scriptNode, ok := lookup(asMapping(withNode), "script"); ok {
					if scriptValue, ok := scalarString(scriptNode); ok && containsUntrustedContext(scriptValue) {
						findings = append(findings, newFinding("CF-INJ-002", file, scriptNode, "Untrusted GitHub context is passed into actions/github-script.", firstUntrustedContext(scriptValue), "jobs.*.steps[].with.script"))
					}
				}
			}
		}
		if _, withNode, ok := lookup(step, "with"); ok {
			for _, key := range []string{"args", "arguments", "command", "cmd", "entrypoint"} {
				if _, valueNode, ok := lookup(asMapping(withNode), key); ok {
					if value, ok := scalarString(valueNode); ok && containsUntrustedContext(value) {
						findings = append(findings, newFinding("CF-INJ-003", file, valueNode, "Untrusted context is passed into action command arguments.", firstUntrustedContext(value), "jobs.*.steps[].with."+key))
					}
				}
			}
			if isCacheOrArtifactAction(usesValue) {
				for _, key := range []string{"key", "restore-keys", "name"} {
					if _, valueNode, ok := lookup(asMapping(withNode), key); ok {
						if value, ok := scalarString(valueNode); ok && containsUntrustedContext(value) {
							findings = append(findings, newFinding("CF-INJ-003", file, valueNode, "Untrusted context is used in a cache key or artifact name.", firstUntrustedContext(value), "jobs.*.steps[].with."+key))
							if isCacheAction(usesValue) && (key == "key" || key == "restore-keys") {
								findings = append(findings, newFinding("CF-CACHE-001", file, valueNode, "Cache key material is derived from attacker-controlled workflow context.", firstUntrustedContext(value), "jobs.*.steps[].with."+key))
							}
						}
					}
				}
			}
		}
	}
	return findings
}

func workflowRunBoundaryFindings(file string, root *yaml.Node) []githubactions.Finding {
	if !hasEvent(root, "workflow_run") {
		return nil
	}
	var findings []githubactions.Finding
	workflowPermissions := permissionSet(mustLookup(root, "permissions"))
	if hasDangerousWrite(workflowPermissions) {
		findings = append(findings, newFinding("CF-RUN-001", file, mustLookup(root, "permissions"), "workflow_run grants dangerous write permissions across a workflow boundary.", "workflow_run write token", "permissions"))
	}
	for _, job := range jobs(root) {
		jobMap := asMapping(job.Value)
		if jobMap == nil {
			continue
		}
		if hasDangerousWrite(permissionSet(mustLookup(jobMap, "permissions"))) {
			message := fmt.Sprintf("workflow_run job %q grants dangerous write permissions across a workflow boundary.", job.Key.Value)
			findings = append(findings, newFinding("CF-RUN-001", file, job.Key, message, "workflow_run job write token", fmt.Sprintf("jobs.%s.permissions", job.Key.Value)))
		}
		if jobDownloadsArtifact(jobMap) && jobExecutesDownloadedContent(jobMap) {
			message := fmt.Sprintf("workflow_run job %q downloads artifacts and executes shell content.", job.Key.Value)
			findings = append(findings, newFinding("CF-ART-001", file, job.Key, message, "workflow_run artifact execution", fmt.Sprintf("jobs.%s", job.Key.Value)))
		}
	}
	return findings
}

func selfHostedRunnerFindings(file string, root *yaml.Node) []githubactions.Finding {
	if !hasUntrustedTrigger(root) {
		return nil
	}
	var findings []githubactions.Finding
	for _, job := range jobs(root) {
		jobMap := asMapping(job.Value)
		if jobMap == nil {
			continue
		}
		_, runsOnNode, ok := lookup(jobMap, "runs-on")
		if !ok || !runsOnSelfHosted(runsOnNode) {
			continue
		}
		message := fmt.Sprintf("Job %q uses a self-hosted runner on an untrusted trigger.", job.Key.Value)
		findings = append(findings, newFinding("CF-RUNNER-001", file, runsOnNode, message, "self-hosted runner", fmt.Sprintf("jobs.%s.runs-on", job.Key.Value)))
	}
	return findings
}

func pullRequestTargetFindings(file string, root *yaml.Node) []githubactions.Finding {
	if !hasPullRequestTarget(root) {
		return nil
	}

	var findings []githubactions.Finding
	if hasDangerousWrite(permissionSet(mustLookup(root, "permissions")), nil) {
		findings = append(findings, newFinding("CF-TRG-004", file, mustLookup(root, "permissions"), "pull_request_target workflow grants a dangerous write permission.", "pull_request_target write token", "permissions"))
	}
	for _, job := range jobs(root) {
		jobMap := asMapping(job.Value)
		if jobMap == nil {
			continue
		}
		if hasDangerousWrite(permissionSet(mustLookup(jobMap, "permissions")), nil) {
			message := fmt.Sprintf("pull_request_target job %q grants a dangerous write permission.", job.Key.Value)
			findings = append(findings, newFinding("CF-TRG-004", file, job.Key, message, "pull_request_target job write token", fmt.Sprintf("jobs.%s.permissions", job.Key.Value)))
		}
	}
	for _, step := range steps(root) {
		if _, runNode, ok := lookup(step, "run"); ok {
			runValue, _ := scalarString(runNode)
			evidence := "run step"
			if pattern := firstShellPattern(runValue); pattern != "" {
				evidence = pattern
			}
			findings = append(findings, newFinding("CF-TRG-002", file, runNode, "pull_request_target workflow executes a run step.", evidence, "jobs.*.steps[].run"))
		}
		_, usesNode, ok := lookup(step, "uses")
		if !ok {
			continue
		}
		usesValue, ok := scalarString(usesNode)
		if !ok {
			continue
		}
		if isThirdPartyAction(usesValue) {
			findings = append(findings, newFinding("CF-TRG-003", file, usesNode, "pull_request_target workflow uses a third-party action.", usesValue, "jobs.*.steps[].uses"))
		}
		if isCacheOrArtifactAction(usesValue) {
			if _, withNode, ok := lookup(step, "with"); ok {
				for _, key := range []string{"key", "restore-keys", "name"} {
					if _, valueNode, ok := lookup(asMapping(withNode), key); ok {
						if value, ok := scalarString(valueNode); ok && containsUntrustedContext(value) {
							findings = append(findings, newFinding("CF-TRG-005", file, valueNode, "pull_request_target cache or artifact data uses PR-controlled context.", firstUntrustedContext(value), "jobs.*.steps[].with."+key))
						}
					}
				}
			}
		}
		if !isCheckoutAction(usesValue) {
			continue
		}
		_, withNode, ok := lookup(step, "with")
		if !ok {
			continue
		}
		_, refNode, ok := lookup(asMapping(withNode), "ref")
		if !ok {
			continue
		}
		refValue, ok := scalarString(refNode)
		if !ok {
			continue
		}
		if containsUnsafeCheckoutRef(refValue) {
			findings = append(findings, newFinding("CF-TRG-001", file, refNode, "pull_request_target workflow checks out untrusted pull request code.", refValue))
		}
	}
	return findings
}

func containerImageFindings(file string, root *yaml.Node) []githubactions.Finding {
	var findings []githubactions.Finding
	for _, job := range jobs(root) {
		jobMap := asMapping(job.Value)
		if jobMap == nil {
			continue
		}
		if _, containerNode, ok := lookup(jobMap, "container"); ok {
			if imageNode, image := containerImage(containerNode); image != "" {
				if imageUsesLatest(image) {
					findings = append(findings, newFinding("CF-IMG-003", file, imageNode, "Job container image uses the mutable latest tag.", image, fmt.Sprintf("jobs.%s.container", job.Key.Value)))
				}
				if !imageUsesDigest(image) {
					findings = append(findings, newFinding("CF-IMG-001", file, imageNode, "Job container image is not pinned by digest.", image, fmt.Sprintf("jobs.%s.container", job.Key.Value)))
				}
			}
		}
		if _, servicesNode, ok := lookup(jobMap, "services"); ok {
			for _, service := range mappingPairs(asMapping(servicesNode)) {
				serviceMap := asMapping(service.Value)
				if serviceMap == nil {
					continue
				}
				_, imageNode, ok := lookup(serviceMap, "image")
				if !ok {
					continue
				}
				image, ok := scalarString(imageNode)
				if !ok {
					continue
				}
				if imageUsesLatest(image) {
					findings = append(findings, newFinding("CF-IMG-003", file, imageNode, "Service container image uses the mutable latest tag.", image, fmt.Sprintf("jobs.%s.services.%s.image", job.Key.Value, service.Key.Value)))
				}
				if !imageUsesDigest(image) {
					findings = append(findings, newFinding("CF-IMG-002", file, imageNode, "Service container image is not pinned by digest.", image, fmt.Sprintf("jobs.%s.services.%s.image", job.Key.Value, service.Key.Value)))
				}
			}
		}
	}
	return findings
}

func jobs(root *yaml.Node) []pair {
	_, jobsNode, ok := lookup(root, "jobs")
	if !ok {
		return nil
	}
	return mappingPairs(asMapping(jobsNode))
}

func steps(root *yaml.Node) []*yaml.Node {
	var out []*yaml.Node
	for _, job := range jobs(root) {
		jobMap := asMapping(job.Value)
		if jobMap == nil {
			continue
		}
		_, stepsNode, ok := lookup(jobMap, "steps")
		if !ok {
			continue
		}
		stepSequence := asSequence(stepsNode)
		if stepSequence == nil {
			continue
		}
		for _, stepNode := range stepSequence.Content {
			if stepMap := asMapping(stepNode); stepMap != nil {
				out = append(out, stepMap)
			}
		}
	}
	return out
}

func hasPullRequestTarget(root *yaml.Node) bool {
	return hasEvent(root, "pull_request_target")
}

func hasEvent(root *yaml.Node, event string) bool {
	_, onNode, ok := lookup(root, "on")
	return ok && nodeContainsEvent(onNode, event)
}

func nodeContainsEvent(node *yaml.Node, event string) bool {
	if node == nil {
		return false
	}
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Value == event
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if item.Value == event {
				return true
			}
		}
	case yaml.MappingNode:
		_, _, ok := lookup(node, event)
		return ok
	}
	return false
}

func newFinding(ruleID string, file string, node *yaml.Node, message string, evidence string, yamlPath ...string) githubactions.Finding {
	definition := DefinitionByID(ruleID)
	line, column := 1, 1
	if node != nil {
		line = maxInt(1, node.Line)
		column = maxInt(1, node.Column)
	}
	return githubactions.Finding{
		RuleID:      definition.ID,
		Severity:    definition.Severity,
		Title:       definition.Title,
		Message:     message,
		File:        file,
		Line:        line,
		Column:      column,
		YAMLPath:    optionalYAMLPath(yamlPath),
		Evidence:    sanitizeEvidence(evidence),
		Remediation: definition.Remediation,
	}
}

func optionalYAMLPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func isWriteAll(node *yaml.Node) bool {
	value, ok := scalarString(node)
	return ok && value == "write-all"
}

func isNullNode(node *yaml.Node) bool {
	return node == nil || node.Tag == "!!null"
}

func permissionSet(node *yaml.Node) map[string]string {
	out := map[string]string{}
	value, ok := scalarString(node)
	if ok {
		lower := strings.ToLower(value)
		if lower == "write-all" {
			out["*"] = "write"
		}
		if lower == "read-all" {
			out["*"] = "read"
		}
		return out
	}
	for _, item := range mappingPairs(asMapping(node)) {
		scope := strings.ToLower(strings.TrimSpace(item.Key.Value))
		permission, ok := scalarString(item.Value)
		if !ok {
			continue
		}
		out[scope] = strings.ToLower(permission)
	}
	return out
}

func unknownPermissionFindings(file string, node *yaml.Node, permissions map[string]string, label string, yamlPath string) []githubactions.Finding {
	var findings []githubactions.Finding
	for scope := range permissions {
		if scope == "*" {
			continue
		}
		if _, ok := knownPermissionScopes[scope]; ok {
			continue
		}
		message := fmt.Sprintf("%s declares unknown GitHub token permission scope %q.", label, scope)
		findings = append(findings, newFinding("CF-PERM-007", file, node, message, scope, yamlPath+"."+scope))
	}
	return findings
}

func dangerousPermissionFindings(file string, node *yaml.Node, permissions map[string]string, label string, ruleID string, yamlPath string) []githubactions.Finding {
	var findings []githubactions.Finding
	if permissions["*"] == "write" {
		message := fmt.Sprintf("%s grants write-all on a pull request-like event.", label)
		findings = append(findings, newFinding(ruleID, file, node, message, label+" write-all", yamlPath))
		return findings
	}
	for scope, value := range permissions {
		if value != "write" {
			continue
		}
		if _, ok := dangerousWriteScopes[scope]; !ok {
			continue
		}
		message := fmt.Sprintf("%s grants %s: write on a pull request-like event.", label, scope)
		findings = append(findings, newFinding(ruleID, file, node, message, scope+": write", yamlPath+"."+scope))
	}
	return findings
}

func hasIDTokenWrite(permissions map[string]string) bool {
	return permissions["*"] == "write" || permissions["id-token"] == "write"
}

func hasDangerousWrite(permissionSets ...map[string]string) bool {
	for _, permissions := range permissionSets {
		if permissions["*"] == "write" {
			return true
		}
		for scope, value := range permissions {
			if value == "write" {
				if _, ok := dangerousWriteScopes[scope]; ok {
					return true
				}
			}
		}
	}
	return false
}

func escalatesPermissions(workflowPermissions map[string]string, jobPermissions map[string]string) bool {
	if len(jobPermissions) == 0 || jobPermissions["*"] == "read" {
		return false
	}
	if jobPermissions["*"] == "write" && workflowPermissions["*"] != "write" {
		return true
	}
	for scope, value := range jobPermissions {
		if value != "write" {
			continue
		}
		if _, ok := dangerousWriteScopes[scope]; !ok {
			continue
		}
		if workflowPermissions["*"] != "write" && workflowPermissions[scope] != "write" {
			return true
		}
	}
	return false
}

func hasJobEnvironment(jobMap *yaml.Node) bool {
	_, _, ok := lookup(jobMap, "environment")
	return ok
}

func hasTrustedPushBranch(root *yaml.Node) bool {
	_, onNode, ok := lookup(root, "on")
	if !ok {
		return false
	}
	if onNode.Kind != yaml.MappingNode {
		return false
	}
	_, pushNode, ok := lookup(onNode, "push")
	if !ok {
		return false
	}
	pushMap := asMapping(pushNode)
	if pushMap == nil {
		return false
	}
	_, branchesNode, ok := lookup(pushMap, "branches")
	if !ok {
		return false
	}
	for _, branch := range scalarValues(branchesNode) {
		if !isTrustedBranch(branch) {
			return false
		}
	}
	return len(scalarValues(branchesNode)) > 0
}

func scalarValues(node *yaml.Node) []string {
	if value, ok := scalarString(node); ok {
		return []string{value}
	}
	sequence := asSequence(node)
	if sequence == nil {
		return nil
	}
	values := make([]string, 0, len(sequence.Content))
	for _, item := range sequence.Content {
		if value, ok := scalarString(item); ok {
			values = append(values, value)
		}
	}
	return values
}

func isTrustedBranch(branch string) bool {
	branch = strings.ToLower(strings.TrimSpace(branch))
	return branch == "main" || branch == "master" || strings.HasPrefix(branch, "release/")
}

func jobHasThirdPartyAction(jobMap *yaml.Node) bool {
	for _, step := range stepsOfJob(jobMap) {
		if _, usesNode, ok := lookup(step, "uses"); ok {
			if usesValue, ok := scalarString(usesNode); ok && isThirdPartyAction(usesValue) {
				return true
			}
		}
	}
	return false
}

func stepsOfJob(jobMap *yaml.Node) []*yaml.Node {
	_, stepsNode, ok := lookup(jobMap, "steps")
	if !ok {
		return nil
	}
	stepSequence := asSequence(stepsNode)
	if stepSequence == nil {
		return nil
	}
	out := make([]*yaml.Node, 0, len(stepSequence.Content))
	for _, stepNode := range stepSequence.Content {
		if stepMap := asMapping(stepNode); stepMap != nil {
			out = append(out, stepMap)
		}
	}
	return out
}

func isLocalAction(usesValue string) bool {
	return strings.HasPrefix(usesValue, "./") || strings.HasPrefix(usesValue, "../") || strings.HasPrefix(usesValue, "/")
}

func splitActionRef(usesValue string) (string, string, bool) {
	index := strings.LastIndex(usesValue, "@")
	if index < 0 {
		return usesValue, "", false
	}
	return usesValue[:index], usesValue[index+1:], true
}

func isMutableRef(ref string) bool {
	_, ok := mutableRefs[strings.ToLower(strings.TrimSpace(ref))]
	return ok
}

func dockerUsesLatest(usesValue string) bool {
	lower := strings.ToLower(strings.TrimSpace(usesValue))
	return strings.HasSuffix(lower, ":latest") && !strings.Contains(lower, "@sha256:")
}

func dockerUsesDigest(usesValue string) bool {
	return strings.Contains(strings.ToLower(usesValue), "@sha256:")
}

func isCheckoutAction(usesValue string) bool {
	action, _, _ := splitActionRef(strings.ToLower(usesValue))
	return action == "actions/checkout" || strings.HasSuffix(action, "/checkout")
}

func isGitHubScriptAction(usesValue string) bool {
	action, _, _ := splitActionRef(strings.ToLower(usesValue))
	return action == "actions/github-script" || strings.HasSuffix(action, "/github-script")
}

func isCacheOrArtifactAction(usesValue string) bool {
	action, _, _ := splitActionRef(strings.ToLower(usesValue))
	return action == "actions/cache" || action == "actions/upload-artifact" || action == "actions/download-artifact" || strings.HasSuffix(action, "/cache") || strings.HasSuffix(action, "/upload-artifact") || strings.HasSuffix(action, "/download-artifact")
}

func isCacheAction(usesValue string) bool {
	action, _, _ := splitActionRef(strings.ToLower(usesValue))
	return action == "actions/cache" || strings.HasSuffix(action, "/cache")
}

func isDownloadArtifactAction(usesValue string) bool {
	action, _, _ := splitActionRef(strings.ToLower(usesValue))
	return action == "actions/download-artifact" || strings.HasSuffix(action, "/download-artifact")
}

func isThirdPartyAction(usesValue string) bool {
	lower := strings.ToLower(strings.TrimSpace(usesValue))
	if lower == "" || isLocalAction(lower) || strings.HasPrefix(lower, "docker://") {
		return false
	}
	action, _, _ := splitActionRef(lower)
	owner, _, ok := strings.Cut(action, "/")
	if !ok {
		return false
	}
	return owner != "actions" && owner != "github"
}

func containsUntrustedContext(value string) bool {
	return firstUntrustedContext(value) != ""
}

func firstUntrustedContext(value string) string {
	lower := strings.ToLower(value)
	for _, contextRef := range untrustedContextRefs {
		if strings.Contains(lower, contextRef) {
			return contextRef
		}
	}
	return ""
}

func containsUnsafeCheckoutRef(value string) bool {
	lower := strings.ToLower(value)
	for _, unsafeRef := range unsafeCheckoutRefs {
		if strings.Contains(lower, unsafeRef) {
			return true
		}
	}
	return false
}

func firstShellPattern(value string) string {
	lower := strings.ToLower(value)
	for _, pattern := range pullRequestTargetShellPatterns {
		if strings.Contains(lower, pattern) {
			return pattern
		}
	}
	return ""
}

func containsEnvironmentFileSink(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "github_env") || strings.Contains(lower, "github_output") || strings.Contains(lower, "github_path")
}

func hasUntrustedTrigger(root *yaml.Node) bool {
	if hasEvent(root, "pull_request") || hasPullRequestTarget(root) || hasEvent(root, "issues") || hasEvent(root, "issue_comment") || hasEvent(root, "discussion") || hasEvent(root, "workflow_run") {
		return true
	}
	return hasEvent(root, "push") && !hasTrustedPushBranch(root)
}

func runsOnSelfHosted(node *yaml.Node) bool {
	if value, ok := scalarString(node); ok {
		return strings.Contains(strings.ToLower(value), "self-hosted")
	}
	for _, value := range scalarValues(node) {
		if strings.EqualFold(strings.TrimSpace(value), "self-hosted") {
			return true
		}
	}
	return false
}

func jobDownloadsArtifact(jobMap *yaml.Node) bool {
	for _, step := range stepsOfJob(jobMap) {
		if _, usesNode, ok := lookup(step, "uses"); ok {
			if usesValue, ok := scalarString(usesNode); ok && isDownloadArtifactAction(usesValue) {
				return true
			}
		}
	}
	return false
}

func jobExecutesDownloadedContent(jobMap *yaml.Node) bool {
	for _, step := range stepsOfJob(jobMap) {
		_, runNode, ok := lookup(step, "run")
		if !ok {
			continue
		}
		runValue, _ := scalarString(runNode)
		lower := strings.ToLower(runValue)
		for _, pattern := range []string{"bash ", "sh ", "source ", "./", "python ", "node ", "chmod +x"} {
			if strings.Contains(lower, pattern) {
				return true
			}
		}
	}
	return false
}

func containerImage(node *yaml.Node) (*yaml.Node, string) {
	if value, ok := scalarString(node); ok {
		return node, value
	}
	if _, imageNode, ok := lookup(asMapping(node), "image"); ok {
		if value, ok := scalarString(imageNode); ok {
			return imageNode, value
		}
	}
	return node, ""
}

func imageUsesDigest(image string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(image)), "@sha256:")
}

func imageUsesLatest(image string) bool {
	lower := strings.ToLower(strings.TrimSpace(image))
	if strings.Contains(lower, "@sha256:") {
		return false
	}
	return strings.HasSuffix(lower, ":latest") || lower == "latest"
}

func isScalarValue(node *yaml.Node, expected string) bool {
	value, ok := scalarString(node)
	return ok && strings.EqualFold(value, expected)
}

func mustLookup(mapping *yaml.Node, key string) *yaml.Node {
	_, value, ok := lookup(mapping, key)
	if !ok {
		return nil
	}
	return value
}

func sanitizeEvidence(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 240 {
		return value[:240]
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
