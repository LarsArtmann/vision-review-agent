package a2ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrDecompile wraps every Decompile failure.
var ErrDecompile = errors.New("a2ui: decompile")

// Decompile is the inverse of Compile: it reconstructs the SurfaceSpec a
// message sequence was compiled from. It accepts exactly one surface
// lifecycle: one createSurface, any number of updateComponents (the last
// wins, matching surface-replacement semantics) and updateDataModel writes
// (folded in order), and no deleteSurface (a destroyed surface has no spec).
//
// Limits inherited from SurfaceSpec: dynamic child lists and createSurface
// fields without a spec counterpart (SendDataModel) have no representation
// and fail with ErrDecompile.
func Decompile(messages []Message) (SurfaceSpec, error) {
	var spec SurfaceSpec

	created := false
	updated := false

	for index, msg := range messages {
		if err := decompileMessage(&spec, &created, index, msg); err != nil {
			return SurfaceSpec{}, err
		}

		if _, isUpdate := msg.(*UpdateComponents); isUpdate {
			updated = true
		}
	}

	if !created {
		return SurfaceSpec{}, fmt.Errorf("%w: no createSurface in %d messages", ErrDecompile, len(messages))
	}

	if !updated {
		return SurfaceSpec{}, fmt.Errorf("%w: no updateComponents in %d messages", ErrDecompile, len(messages))
	}

	return spec, nil
}

// decompileMessage folds one message into the spec under construction.
func decompileMessage(spec *SurfaceSpec, created *bool, index int, msg Message) error {
	if msg == nil {
		return fmt.Errorf("%w: message %d is nil", ErrDecompile, index)
	}

	switch typed := msg.(type) {
	case *CreateSurface:
		if *created {
			return fmt.Errorf(
				"%w: second createSurface at message %d (surface %q)",
				ErrDecompile, index, typed.SurfaceID,
			)
		}

		if typed.SendDataModel {
			return fmt.Errorf(
				"%w: sendDataModel has no SurfaceSpec representation (message %d)",
				ErrDecompile, index,
			)
		}

		spec.SurfaceID = typed.SurfaceID
		spec.CatalogID = typed.CatalogID
		spec.Theme = typed.Theme
		*created = true
	case *UpdateComponents:
		if !*created {
			return fmt.Errorf(
				"%w: updateComponents before createSurface at message %d", ErrDecompile, index,
			)
		}

		specs, err := specsFromWire(typed.Components)
		if err != nil {
			return fmt.Errorf("%w: message %d: %w", ErrDecompile, index, err)
		}

		spec.Components = specs
	case *UpdateDataModel:
		if !*created {
			return fmt.Errorf(
				"%w: updateDataModel before createSurface at message %d", ErrDecompile, index,
			)
		}

		if err := applyDataModel(spec, typed); err != nil {
			return fmt.Errorf("%w: message %d: %w", ErrDecompile, index, err)
		}
	case *DeleteSurface:
		return fmt.Errorf(
			"%w: surface %q was deleted at message %d; a destroyed surface has no spec",
			ErrDecompile, typed.SurfaceID, index,
		)
	default:
		return fmt.Errorf("%w: unknown message %T at %d", ErrDecompile, msg, index)
	}

	return nil
}

// specsFromWire maps wire components back to their spec shape, rejecting the
// shapes SurfaceSpec cannot express.
func specsFromWire(components []Component) ([]ComponentSpec, error) {
	specs := make([]ComponentSpec, 0, len(components))

	for _, component := range components {
		spec := ComponentSpec{
			ID:            component.ID,
			Kind:          component.Kind,
			Accessibility: component.Accessibility,
			Properties:    component.Props,
		}

		if component.Child != nil {
			spec.Child = *component.Child
		}

		switch {
		case component.Children == nil:
		case component.Children.Dynamic != nil:
			return nil, fmt.Errorf(
				"%w: component %q: dynamic child lists have no SurfaceSpec representation; edit the wire messages",
				ErrDecompile, component.ID,
			)
		default:
			spec.Children = component.Children.Static
		}

		specs = append(specs, spec)
	}

	return specs, nil
}

// applyDataModel folds one updateDataModel message into the spec's data
// model, honoring whole-model replacement, JSON-Pointer writes, and removes.
func applyDataModel(spec *SurfaceSpec, update *UpdateDataModel) error {
	switch {
	case update.Remove && update.Path == "":
		spec.DataModel = nil

		return nil
	case update.Remove:
		if spec.DataModel == nil {
			return nil
		}

		return removeAtPointer(spec.DataModel, update.Path)
	case update.Path == "":
		whole, ok := update.Value.(map[string]any)
		if !ok {
			return fmt.Errorf("%w: whole-model value is %T, want object", ErrDecompile, update.Value)
		}

		spec.DataModel = whole

		return nil
	default:
		if spec.DataModel == nil {
			spec.DataModel = map[string]any{}
		}

		return setAtPointer(spec.DataModel, update.Path, update.Value)
	}
}

// pointerTokens splits an RFC 6901 JSON Pointer into decoded tokens.
func pointerTokens(path string) ([]string, error) {
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("%w: path %q is not a JSON Pointer (must start with \"/\")", ErrDecompile, path)
	}

	raw := strings.Split(path[1:], "/")
	tokens := make([]string, len(raw))

	for i, token := range raw {
		tokens[i] = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
	}

	return tokens, nil
}

// setAtPointer writes value at an RFC 6901 pointer inside root, creating
// intermediate objects as needed.
func setAtPointer(root map[string]any, path string, value any) error {
	tokens, err := pointerTokens(path)
	if err != nil {
		return err
	}

	container := any(root)

	for depth, token := range tokens[:len(tokens)-1] {
		next, err := descendOrCreate(container, token)
		if err != nil {
			return fmt.Errorf("token %d (%q): %w", depth, token, err)
		}

		container = next
	}

	leaf := tokens[len(tokens)-1]

	switch parent := container.(type) {
	case map[string]any:
		parent[leaf] = value
	case []any:
		index, err := indexAt(parent, leaf)
		if err != nil {
			return err
		}

		parent[index] = value
	default:
		return fmt.Errorf("%w: cannot index %T with %q", ErrDecompile, container, leaf)
	}

	return nil
}

// removeAtPointer deletes the value at an RFC 6901 pointer inside root.
func removeAtPointer(root map[string]any, path string) error {
	tokens, err := pointerTokens(path)
	if err != nil {
		return err
	}

	container := any(root)

	for _, token := range tokens[:len(tokens)-1] {
		next, err := descendOnly(container, token)
		if err != nil {
			return err
		}

		container = next
	}

	leaf := tokens[len(tokens)-1]

	switch parent := container.(type) {
	case map[string]any:
		delete(parent, leaf)
	case []any:
		index, err := indexAt(parent, leaf)
		if err != nil {
			return err
		}

		parent[index] = nil
	default:
		return fmt.Errorf("%w: cannot index %T with %q", ErrDecompile, container, leaf)
	}

	return nil
}

// descendOrCreate walks one pointer token, creating missing objects.
func descendOrCreate(container any, token string) (any, error) {
	switch node := container.(type) {
	case map[string]any:
		child, ok := node[token]
		if !ok || child == nil {
			child = map[string]any{}
			node[token] = child
		}

		return child, nil
	case []any:
		index, err := indexAt(node, token)
		if err != nil {
			return nil, err
		}

		return node[index], nil
	default:
		return nil, fmt.Errorf("%w: cannot descend into scalar %T", ErrDecompile, container)
	}
}

// descendOnly walks one pointer token without creating anything.
func descendOnly(container any, token string) (any, error) {
	switch node := container.(type) {
	case map[string]any:
		return node[token], nil
	case []any:
		index, err := indexAt(node, token)
		if err != nil {
			return nil, err
		}

		return node[index], nil
	default:
		return nil, fmt.Errorf("%w: cannot descend into scalar %T", ErrDecompile, container)
	}
}

// indexAt resolves an array index token.
func indexAt(array []any, token string) (int, error) {
	index, err := strconv.Atoi(token)
	if err != nil || index < 0 || index >= len(array) {
		return 0, fmt.Errorf("%w: index %q out of range for array of %d", ErrDecompile, token, len(array))
	}

	return index, nil
}
