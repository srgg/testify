package depend

// Dep represents test method dependencies.
// Maps child test names to their required parent tests.
type Dep struct {
	Deps map[string][]string
}

// Depends constructs a Dep by invoking fn.
// Used by generated code to declare test dependencies.
func Depends(fn func(any) *Dep) *Dep {
	return fn(nil)
}

// On declares that the child depends on the parents.
// Child test runs only if all parents pass.
// Returns self for chaining.
func (d *Dep) On(child string, parents ...string) *Dep {
	if d.Deps == nil {
		d.Deps = map[string][]string{}
	}
	d.Deps[child] = parents
	return d
}
