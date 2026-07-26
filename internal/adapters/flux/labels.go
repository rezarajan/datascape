package flux

// labelDomain is the identifier stem every d7s-emitted label lives under
// (CLAUDE.md naming decision, week-one plan).
const labelDomain = "d7s.dev"

// ownershipLabels returns the labels every d7s-emitted object carries
// (golden rule 27): who manages it, which stack it belongs to, and
// which component within that stack — component is omitted for
// stack-level or platform-level objects that aren't owned by one
// component.
func ownershipLabels(stack, component string) map[string]string {
	labels := map[string]string{
		labelDomain + "/managed-by": "d7s",
	}
	if stack != "" {
		labels[labelDomain+"/stack"] = stack
	}
	if component != "" {
		labels[labelDomain+"/component"] = component
	}
	return labels
}
