package cmo

// Edition is one year/competition entry in the CMO or CJMO archive.
type Edition struct {
	Rank         int    `json:"rank"          csv:"rank"          tsv:"rank"`
	Year         int    `json:"year"          csv:"year"          tsv:"year"`
	Competition  string `json:"competition"   csv:"competition"   tsv:"competition"`
	ProblemsURL  string `json:"problems_url"  csv:"problems_url"  tsv:"problems_url"`
	SolutionsURL string `json:"solutions_url" csv:"solutions_url" tsv:"solutions_url"`
}
