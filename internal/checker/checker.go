package checker

import "context"

// Checker represents an update checker
type Checker interface {
	Check(ctx context.Context) (bool, error)
}

// Named represents a checker with a name
type Named struct {
	Name    string
	Checker Checker
}
