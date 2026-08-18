package a2ui

import (
	"errors"
	"fmt"
	"strings"
)

// ErrValidation wraps every Validate failure; inspect the chained errors (or
// use Issues) for the individual structural problems.
var ErrValidation = errors.New("a2ui: message stream failed structural validation")

// ErrInvalidMessage wraps every per-message structural problem reported by
// Validate and Issues.
var ErrInvalidMessage = errors.New("a2ui: invalid message")

// ErrComponentCycle is reported (wrapped, with the cycle path) when the
// component graph reachable from root contains a reference cycle.
var ErrComponentCycle = errors.New("a2ui: component reference cycle")

// ValidationIssue is one structural validation problem, located by message
// index in the validated stream. It is plain data; Validate turns issues
// into errors wrapping ErrInvalidMessage (or the issue's typed Err).
type ValidationIssue struct {
	// MessageIndex is the position in the validated stream.
	MessageIndex int

	// Detail describes the problem; used when Err is nil.
	Detail string

	// Err is an optional typed cause (e.g. ErrComponentCycle) that Validate
	// wraps so errors.Is matches.
	Err error
}

// Validate checks a message stream against the structural rules of the A2UI
// v0.9.x protocol: envelope versions, surface lifecycle ordering, required
// fields, root component presence, unique component IDs, resolvable child
// references, and acyclicity. Component reachability from root is NOT
// required (the spec permits unreferenced definitions).
//
// It returns nil, or an error wrapping ErrValidation (and, where applicable,
// typed causes like ErrComponentCycle) with one issue per line.
func Validate(messages []Message) error {
	issues := Issues(messages)
	if len(issues) == 0 {
		return nil
	}

	errs := make([]error, len(issues)+1)
	errs[0] = fmt.Errorf("%w: %d issue(s)", ErrValidation, len(issues))

	for i, issue := range issues {
		switch {
		case issue.Err != nil:
			errs[i+1] = fmt.Errorf("message %d: %w", issue.MessageIndex, issue.Err)
		default:
			errs[i+1] = fmt.Errorf("%w at index %d: %s", ErrInvalidMessage, issue.MessageIndex, issue.Detail)
		}
	}

	return errors.Join(errs...)
}

// Issues runs the same checks as Validate but returns every problem instead
// of an error, so callers can report or triage them individually. An empty
// slice means the stream is structurally valid.
func Issues(messages []Message) []ValidationIssue {
	issues := make([]ValidationIssue, 0, len(messages))
	openSurfaces := make(map[string]string) // surfaceID -> catalogID

	for index, msg := range messages {
		if msg == nil {
			issues = append(issues, ValidationIssue{MessageIndex: index, Detail: "message is nil"})

			continue
		}

		issues = append(issues, validateEnvelope(index, msg)...)

		switch typed := msg.(type) {
		case *CreateSurface:
			trackCreateSurface(&issues, index, typed, openSurfaces)
		case *UpdateComponents:
			issues = append(
				issues,
				validateOpenSurface(index, kindUpdateComponents, typed.SurfaceID, openSurfaces)...)
			issues = append(issues, validateComponents(index, typed.Components)...)
		case *UpdateDataModel:
			issues = append(issues, validateDataModel(index, typed, openSurfaces)...)
		case *DeleteSurface:
			trackDeleteSurface(&issues, index, typed, openSurfaces)
		}
	}

	return issues
}

// validateEnvelope checks the fields every message kind shares: version and
// surfaceId.
func validateEnvelope(index int, msg Message) []ValidationIssue {
	var issues []ValidationIssue

	add := func(format string, args ...any) {
		issues = append(issues, ValidationIssue{MessageIndex: index, Detail: fmt.Sprintf(format, args...)})
	}

	switch version := msg.Version(); {
	case version == "":
		add("missing envelope version")
	case !isKnownVersion(version):
		add("unknown envelope version %q (known: %s, %s)", version, VersionV09, VersionV091)
	}

	if msg.Surface() == "" {
		add("empty surfaceId")
	}

	return issues
}

// trackCreateSurface records a created surface, rejecting duplicates.
func trackCreateSurface(issues *[]ValidationIssue, index int, msg *CreateSurface, openSurfaces map[string]string) {
	if msg.CatalogID == "" {
		*issues = append(*issues, ValidationIssue{
			MessageIndex: index,
			Detail:       "createSurface: catalogId is required",
		})
	}

	if _, exists := openSurfaces[msg.SurfaceID]; exists {
		*issues = append(*issues, ValidationIssue{
			MessageIndex: index,
			Detail:       fmt.Sprintf("createSurface: surface %q already exists (delete it first)", msg.SurfaceID),
		})

		return
	}

	openSurfaces[msg.SurfaceID] = msg.CatalogID
}

// trackDeleteSurface forgets a deleted surface, rejecting unknown ones.
func trackDeleteSurface(issues *[]ValidationIssue, index int, msg *DeleteSurface, openSurfaces map[string]string) {
	if !surfaceExists(openSurfaces, msg.SurfaceID) {
		*issues = append(*issues, ValidationIssue{
			MessageIndex: index,
			Detail:       fmt.Sprintf("deleteSurface: surface %q was never created", msg.SurfaceID),
		})

		return
	}

	delete(openSurfaces, msg.SurfaceID)
}

// validateOpenSurface reports when a message targets a surface that has not
// been created (or was already deleted).
func validateOpenSurface(
	index int,
	kind, surfaceID string,
	openSurfaces map[string]string,
) []ValidationIssue {
	if surfaceExists(openSurfaces, surfaceID) {
		return nil
	}

	return []ValidationIssue{{
		MessageIndex: index,
		Detail:       fmt.Sprintf("%s: surface %q was never created", kind, surfaceID),
	}}
}

// validateDataModel checks an updateDataModel message against the open
// surfaces.
func validateDataModel(index int, msg *UpdateDataModel, openSurfaces map[string]string) []ValidationIssue {
	var issues []ValidationIssue

	issues = append(issues, validateOpenSurface(index, kindUpdateDataModel, msg.SurfaceID, openSurfaces)...)

	if msg.Path != "" && !strings.HasPrefix(msg.Path, "/") {
		issues = append(issues, ValidationIssue{
			MessageIndex: index,
			Detail:       fmt.Sprintf("updateDataModel: path %q must be a JSON Pointer starting with \"/\"", msg.Path),
		})
	}

	return issues
}

// validateComponents checks one updateComponents payload: non-empty list,
// unique non-empty IDs, exactly one root, unambiguous container shape,
// resolvable references, and acyclicity.
func validateComponents(index int, components []Component) []ValidationIssue {
	if len(components) == 0 {
		return []ValidationIssue{{
			MessageIndex: index,
			Detail:       "updateComponents: components list is empty",
		}}
	}

	var issues []ValidationIssue

	byID := indexComponents(&issues, index, components)

	if _, hasRoot := byID[RootID]; !hasRoot {
		issues = append(issues, ValidationIssue{
			MessageIndex: index,
			Detail:       fmt.Sprintf("updateComponents: no component with id %q", RootID),
		})
	}

	for i := range components {
		issues = append(issues, validateComponent(index, &components[i], byID)...)
	}

	if root := byID[RootID]; root != nil {
		if err := detectCycle(root, byID); err != nil {
			issues = append(issues, ValidationIssue{MessageIndex: index, Err: err})
		}
	}

	return issues
}

// indexComponents builds the ID lookup for one components list, flagging
// empty and duplicate IDs.
func indexComponents(issues *[]ValidationIssue, index int, components []Component) map[string]*Component {
	byID := make(map[string]*Component, len(components))

	for i := range components {
		comp := &components[i]

		switch {
		case comp.ID == "":
			*issues = append(*issues, ValidationIssue{
				MessageIndex: index,
				Detail:       fmt.Sprintf("component %d: empty id", i),
			})
		case byID[comp.ID] != nil:
			*issues = append(*issues, ValidationIssue{
				MessageIndex: index,
				Detail:       fmt.Sprintf("component %q: duplicate id", comp.ID),
			})
		default:
			byID[comp.ID] = comp
		}
	}

	return byID
}

// validateComponent checks a single component: type presence, unambiguous
// container shape, and resolvable references.
func validateComponent(index int, comp *Component, byID map[string]*Component) []ValidationIssue {
	var issues []ValidationIssue

	add := func(format string, args ...any) {
		issues = append(issues, ValidationIssue{MessageIndex: index, Detail: fmt.Sprintf(format, args...)})
	}

	if comp.Kind == "" {
		add("component %q: empty component type", comp.ID)
	}

	if comp.Child != nil && comp.Children != nil {
		add("component %q: both child and children set", comp.ID)
	}

	if comp.Child != nil && byID[*comp.Child] == nil {
		add("component %q: child %q is not defined", comp.ID, *comp.Child)
	}

	if comp.Children != nil {
		issues = append(issues, validateChildList(index, comp, comp.Children, byID)...)
	}

	return issues
}

// validateChildList checks one children list: exactly one shape, and every
// static reference resolves.
func validateChildList(index int, comp *Component, list *ChildList, byID map[string]*Component) []ValidationIssue {
	var issues []ValidationIssue

	add := func(format string, args ...any) {
		issues = append(issues, ValidationIssue{MessageIndex: index, Detail: fmt.Sprintf(format, args...)})
	}

	switch {
	case list.Static == nil && list.Dynamic == nil:
		add("component %q: children is neither a static list nor a dynamic template", comp.ID)
	case list.Dynamic != nil:
		if list.Dynamic.ComponentID == "" || list.Dynamic.Path == "" {
			add("component %q: dynamic children need componentId and path", comp.ID)
		}
	default:
		for _, childID := range list.Static {
			if byID[childID] == nil {
				add("component %q: child %q is not defined", comp.ID, childID)
			}
		}
	}

	return issues
}

// detectCycle reports whether the component graph reachable from root
// contains a reference cycle, wrapping ErrComponentCycle with the cycle path.
func detectCycle(root *Component, byID map[string]*Component) error {
	const (
		visiting = 1
		done     = 2
	)

	state := make(map[string]int, len(byID))

	var visit func(comp *Component, trail []string) error

	visit = func(comp *Component, trail []string) error {
		switch state[comp.ID] {
		case done:
			return nil
		case visiting:
			return fmt.Errorf("%w: %s -> %s", ErrComponentCycle, strings.Join(trail, " -> "), comp.ID)
		}

		state[comp.ID] = visiting
		trail = append(trail, comp.ID)

		for _, next := range componentRefs(comp, byID) {
			if err := visit(next, trail); err != nil {
				return err
			}
		}

		state[comp.ID] = done

		return nil
	}

	return visit(root, nil)
}

// componentRefs yields the resolved child references of a component, in
// declaration order.
func componentRefs(comp *Component, byID map[string]*Component) []*Component {
	var refs []*Component

	if comp.Child != nil {
		if next := byID[*comp.Child]; next != nil {
			refs = append(refs, next)
		}
	}

	if comp.Children != nil {
		for _, childID := range comp.Children.Static {
			if next := byID[childID]; next != nil {
				refs = append(refs, next)
			}
		}
	}

	return refs
}

// surfaceExists reports whether the surface is currently open (created and
// not yet deleted).
func surfaceExists(openSurfaces map[string]string, surfaceID string) bool {
	_, ok := openSurfaces[surfaceID]

	return ok
}

// isKnownVersion reports whether version is a protocol version this package
// speaks on the wire.
func isKnownVersion(version string) bool {
	return version == VersionV09 || version == VersionV091
}
