package cmo

import "testing"

func TestExtractPDFLinks(t *testing.T) {
	html := `<html><body>
<a href="https://example.com/foo.pdf">Foo</a>
<a href="https://example.com/bar.html">Bar</a>
<a href="https://example.com/baz.PDF">Baz</a>
</body></html>`
	links := extractPDFLinks([]byte(html))
	if len(links) != 2 {
		t.Fatalf("want 2 PDF links, got %d: %v", len(links), links)
	}
}

func TestClassifyURL(t *testing.T) {
	cases := []struct {
		url         string
		wantYear    int
		wantComp    string
		wantDocType string
	}{
		{
			"https://cms.math.ca/wp-content/uploads/2019/07/exam2010.pdf",
			2010, "CMO", "problems",
		},
		{
			"https://cms.math.ca/wp-content/uploads/2019/07/sol2010.pdf",
			2010, "CMO", "solutions",
		},
		{
			"https://cms.math.ca/wp-content/uploads/2024/03/jexam2020.pdf",
			2020, "CJMO", "problems",
		},
		{
			"https://cms.math.ca/wp-content/uploads/2024/03/jsol2020.pdf",
			2020, "CJMO", "solutions",
		},
		{
			"https://cms.math.ca/wp-content/uploads/2024/03/CJMO2024-problems.pdf",
			2024, "CJMO", "problems",
		},
		{
			"https://cms.math.ca/wp-content/uploads/2024/04/cjmo2024-solutions-en.pdf",
			2024, "CJMO", "solutions",
		},
		{
			"https://cms.math.ca/wp-content/uploads/2025/03/CMO2025-solutions.pdf",
			2025, "CMO", "solutions",
		},
		{
			"https://cms.math.ca/wp-content/uploads/2021/06/2021CMO_solutions_en-1.pdf",
			2021, "CMO", "solutions",
		},
		{
			"https://cms.math.ca/wp-content/uploads/2021/04/CMO-2021-questions-en-4.pdf",
			2021, "CMO", "problems",
		},
	}

	for _, tc := range cases {
		year, comp, docType := classifyURL(tc.url)
		if year != tc.wantYear || comp != tc.wantComp || docType != tc.wantDocType {
			t.Errorf("classifyURL(%q)\n  got  (%d, %q, %q)\n  want (%d, %q, %q)",
				tc.url, year, comp, docType, tc.wantYear, tc.wantComp, tc.wantDocType)
		}
	}
}

func TestBuildEditionsSortOrder(t *testing.T) {
	urls := []string{
		"https://cms.math.ca/wp-content/uploads/2019/07/exam2010.pdf",
		"https://cms.math.ca/wp-content/uploads/2019/07/sol2010.pdf",
		"https://cms.math.ca/wp-content/uploads/2025/03/CMO2025-problems.pdf",
		"https://cms.math.ca/wp-content/uploads/2025/03/CJMO2025-problems.pdf",
		"https://cms.math.ca/wp-content/uploads/2025/04/CMO2025-solutions.pdf",
		"https://cms.math.ca/wp-content/uploads/2025/04/CJMO2025-solutions.pdf",
	}
	eds := buildEditions(urls)
	if len(eds) != 3 {
		t.Fatalf("want 3 editions, got %d", len(eds))
	}
	// 2025 CMO first, then 2025 CJMO, then 2010 CMO.
	if eds[0].Year != 2025 || eds[0].Competition != "CMO" {
		t.Errorf("eds[0] = {%d, %s}, want {2025, CMO}", eds[0].Year, eds[0].Competition)
	}
	if eds[1].Year != 2025 || eds[1].Competition != "CJMO" {
		t.Errorf("eds[1] = {%d, %s}, want {2025, CJMO}", eds[1].Year, eds[1].Competition)
	}
	if eds[2].Year != 2010 {
		t.Errorf("eds[2].Year = %d, want 2010", eds[2].Year)
	}
	// Ranks assigned correctly.
	for i, ed := range eds {
		if ed.Rank != i+1 {
			t.Errorf("eds[%d].Rank = %d, want %d", i, ed.Rank, i+1)
		}
	}
}
