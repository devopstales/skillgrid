package tools

// Tool describes a future installable tool pack.
type Tool struct {
	Name string
}

// Registry is empty in v1.
type Registry struct {
	Tools []Tool
}

func Default() Registry {
	return Registry{Tools: nil}
}
