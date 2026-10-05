package ui

import (
	"slices"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
)

type FormValues struct {
	Text    map[string]string
	Checked map[string]bool
}

type Form struct {
	OnSubmit func(FormValues)
	rt       *twi.Runtime
	fields   []*formField
}

type formField struct {
	key, message string
	control      *control
	node         func(...twi.NodeOption) twi.Node
	open         *bool
	check        func() string
	reset        func()
	put          func(FormValues)
}

func NewForm(rt *twi.Runtime) *Form { return &Form{rt: rt} }

func (f *Form) Input(key string, in *Input, rules ...func(string) string) {
	enter := keyDown(f.rt, func(k input.KeyEvent) bool {
		if k.Key == input.KeyEnter && k.Modifiers == 0 {
			f.Submit()
			return true
		}
		return false
	})
	node := func(options ...twi.NodeOption) twi.Node { return in.Node(append(options, enter)...) }
	register(f, key, &in.control, node, in.Value, in.Set, texts, rules)
}

func (f *Form) Textarea(key string, t *Textarea, rules ...func(string) string) {
	register(f, key, &t.control, t.Node, t.Value, t.Set, texts, rules)
}

func (f *Form) Checkbox(key string, c *Checkbox, rules ...func(bool) string) {
	register(f, key, &c.control, c.Node, func() bool { return c.Checked }, func(on bool) { c.Checked = on }, checks, rules)
}

func (f *Form) Select(key string, s *Select, rules ...func(string) string) {
	register(f, key, &s.control, s.Trigger, func() string { return s.Value }, func(v string) { s.Value = v }, texts, rules).open = &s.Open
}

func texts(v FormValues) map[string]string { return v.Text }

func checks(v FormValues) map[string]bool { return v.Checked }

func register[T any](f *Form, key string, c *control, node func(...twi.NodeOption) twi.Node, get func() T, set func(T), into func(FormValues) map[string]T, rules []func(T) string) *formField {
	start := get()
	fd := &formField{key: key, control: c, node: node,
		check: func() string {
			for _, rule := range rules {
				if message := rule(get()); message != "" {
					return message
				}
			}
			return ""
		},
		reset: func() { set(start) },
		put:   func(v FormValues) { into(v)[key] = get() },
	}
	f.fields = append(f.fields, fd)
	return fd
}

func (f *Form) field(key string) *formField {
	i := slices.IndexFunc(f.fields, func(fd *formField) bool { return fd.key == key })
	if i < 0 {
		panic("ui: form has no field " + key)
	}
	return f.fields[i]
}

func (fd *formField) show(message string) {
	fd.message, fd.control.Invalid = message, message != ""
}

func (f *Form) Control(key string, options ...twi.NodeOption) twi.Node {
	fd := f.field(key)
	blur := twi.OnBlur(func(*twi.Event) {
		if fd.open == nil || !*fd.open {
			fd.show(fd.check())
			f.rt.Invalidate()
		}
	})
	return fd.node(append(options, twi.Key(key), blur)...)
}

func (f *Form) Label(key string, children ...twi.NodeOption) twi.Node {
	if f.field(key).message != "" {
		children = append(children, twi.Class("text-destructive"))
	}
	return FieldLabel(children...)
}

func (f *Form) Item(key string, children ...twi.NodeOption) twi.Node {
	if message := f.field(key).message; message != "" {
		children = append(children, FieldError(twi.Text(message)))
	}
	return Field(Vertical, children...)
}

func (f *Form) Submit() {
	values := FormValues{Text: map[string]string{}, Checked: map[string]bool{}}
	var first *formField
	for _, fd := range f.fields {
		fd.show(fd.check())
		if first == nil && fd.message != "" {
			first = fd
		}
		fd.put(values)
	}
	f.rt.Invalidate()
	if first != nil {
		f.rt.Focus(first.key)
		return
	}
	notify(f.OnSubmit, values)
}

func (f *Form) Reset() {
	for _, fd := range f.fields {
		fd.reset()
		fd.show("")
	}
	f.rt.Invalidate()
}
