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

func Analyze(doc parser.Document) []githubactions.Finding {
	root := documentMapping(doc.Root)
	if root == nil {
		return nil
	}

	findings := make([]githubactions.Finding, 0)
	findings = append(findings, permissionsFindings(doc.File, root)...)
	findings = append(findings, actionReferenceFindings(doc.File, root)...)
	findings = append(findings, pullRequestTargetFindings(doc.File, root)...)

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
		findings = append(findings, newFinding("CF-PERM-001", file, permissionsValue, "Workflow uses permissions: write-all.", "permissions: write-all"))
	}
	if !hasWorkflowPermissions {
		findings = append(findings, newFinding("CF-PERM-002", file, root, "Workflow is missing an explicit permissions block.", "workflow permissions missing"))
	} else if permissionsKey != nil && isNullNode(permissionsValue) {
		findings = append(findings, newFinding("CF-PERM-002", file, permissionsKey, "Workflow permissions block is empty.", "permissions block empty"))
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
			findings = append(findings, newFinding("CF-PERM-001", file, jobPermissionsValue, message, "permissions: write-all"))
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

func pullRequestTargetFindings(file string, root *yaml.Node) []githubactions.Finding {
	if !hasPullRequestTarget(root) {
		return nil
	}

	var findings []githubactions.Finding
	for _, step := range steps(root) {
		_, usesNode, ok := lookup(step, "uses")
		if !ok {
			continue
		}
		usesValue, ok := scalarString(usesNode)
		if !ok || !isCheckoutAction(usesValue) {
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

func steps(root *yaml.Node) []*yaml.Node {
	_, jobsNode, ok := lookup(root, "jobs")
	if !ok {
		return nil
	}
	var out []*yaml.Node
	for _, job := range mappingPairs(asMapping(jobsNode)) {
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
	_, onNode, ok := lookup(root, "on")
	if !ok {
		return false
	}
	return nodeContainsEvent(onNode, "pull_request_target")
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

func newFinding(ruleID string, file string, node *yaml.Node, message string, evidence string) githubactions.Finding {
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
		Evidence:    sanitizeEvidence(evidence),
		Remediation: definition.Remediation,
	}
}

func isWriteAll(node *yaml.Node) bool {
	value, ok := scalarString(node)
	return ok && value == "write-all"
}

func isNullNode(node *yaml.Node) bool {
	return node == nil || node.Tag == "!!null"
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

func containsUnsafeCheckoutRef(value string) bool {
	lower := strings.ToLower(value)
	for _, unsafeRef := range unsafeCheckoutRefs {
		if strings.Contains(lower, unsafeRef) {
			return true
		}
	}
	return false
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
