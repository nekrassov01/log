package log

import "log/slog"

// AttrReplacer replaces each non-group ordinary attribute before formatting.
// It does not receive the built-in time, level, source, label, or message.
// Ordinary attributes with those names are still passed to the replacer.
//
// [slog.LogValuer] values are resolved before and after the call. Returning a
// zero [slog.Attr] removes the attribute. If the result is a group, its children
// are traversed and each non-group child is passed to the replacer.
//
// The groups slice contains the original enclosing group names, outermost first.
// The replacer must not modify or retain this slice. Empty group names are omitted.
// Calls may run concurrently, so the replacer must protect any shared mutable state.
type AttrReplacer func(groups []string, attr slog.Attr) slog.Attr

// attrState tracks original group names for replacement and an escaped,
// dot-separated path for output. Its buffers are reused during traversal.
type attrState struct {
	path   []byte
	groups []string
}

// prepare initializes the group path inherited from a handler.
func (o *attrState) prepare(groups []string) {
	o.reset()
	for _, group := range groups {
		o.pushGroup(group)
	}
}

// resolve resolves and replaces attr, traversing groups depth-first and omitting
// zero attributes. The yielded path aliases the current state and is valid only
// during the callback. Traversal restores the enclosing path before returning.
func (o *attrState) resolve(attr slog.Attr, replacer AttrReplacer, yield func(slog.Attr, slog.Kind, []byte)) {
	kind := attr.Value.Kind()
	if kind == slog.KindLogValuer {
		attr.Value = attr.Value.Resolve()
		kind = attr.Value.Kind()
	}
	if replacer != nil && kind != slog.KindGroup {
		attr = replacer(o.groups, attr)
		kind = attr.Value.Kind()
		if kind == slog.KindLogValuer {
			attr.Value = attr.Value.Resolve()
			kind = attr.Value.Kind()
		}
	}
	if kind == slog.KindGroup {
		key := attr.Key
		pathLen := len(o.path)
		o.pushGroup(key)
		for _, child := range attr.Value.Group() {
			o.resolve(child, replacer, yield)
		}
		o.popGroup(key, pathLen)
		return
	}
	if kind == slog.KindAny && attr.Key == "" && attr.Value.Any() == nil {
		return
	}
	yield(attr, kind, o.path)
}

// pushGroup adds a non-empty group to the current attribute path.
func (o *attrState) pushGroup(group string) {
	if group == "" {
		return
	}
	o.groups = append(o.groups, group)
	o.path, _ = escapeText(o.path, group)
	o.path = append(o.path, '.')
}

// popGroup restores the saved path for a non-empty group.
func (o *attrState) popGroup(group string, pathLen int) {
	if group == "" {
		return
	}
	o.groups = o.groups[:len(o.groups)-1]
	o.path = o.path[:pathLen]
}

// reset clears the current path and active group names while retaining capacity.
func (o *attrState) reset() {
	o.path = o.path[:0]
	clear(o.groups)
	o.groups = o.groups[:0]
}
