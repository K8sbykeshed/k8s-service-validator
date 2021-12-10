package matrix

var MatrixResults *Results

type Results struct {
	results []*Result
}

type Result struct {
	Name string
	label Label
	Result bool
	WrongNum int
}

type Label struct {
	key   string
	value string
}

func (r *Results) Collect(result *Result) {
	r.results = append(r.results, result)
}