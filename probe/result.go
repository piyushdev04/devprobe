package probe

import "fmt"

type Result struct {
	Name     string
	Duration int64 
	Err      error
	Extra    string
	Order    int
}

func (r Result) Format() string {
	if r.Err != nil {
		return fmt.Sprintf("✖ %s failed: %v", r.Name, r.Err)
	}
	if r.Extra != "" {
		return fmt.Sprintf("✔ %s: %dms (%s)", r.Name, r.Duration, r.Extra)
	}
	return fmt.Sprintf("✔ %s: %dms", r.Name, r.Duration)
}