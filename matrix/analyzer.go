package matrix

// nolint
var MatrixResults *Results

// nolint
type Results struct {
	results []*Result
}

// nolint
type Result struct {
	Name     string
	label    Label
	Result   bool
	WrongNum int
}

// nolint
type Label struct {
	key   string
	value string
}

// nolint
func (r *Results) Collect(result *Result) {
	r.results = append(r.results, result)
}
