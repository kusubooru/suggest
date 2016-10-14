package teian

import (
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestSiteTitle(t *testing.T) {
	c := Conf{Title: "a title"}
	want := "A Title"
	if got := c.SiteTitle(); got != want {
		t.Errorf("Conf{Title: %q}.SiteTitle() returned %q, want %q", c.Title, got, want)
	}
}

func TestFmtCreated(t *testing.T) {
	now := time.Now()
	s := Suggestion{Created: now}
	want := now.UTC().Format("Mon 02 Jan 2006 15:04:05 MST")
	if got := s.FmtCreated(); got != want {
		t.Errorf("Suggestion{Created: %q}.FmtCreated() returned \n%q, want \n%q", s.Created, got, want)
	}
}

var findByIDTests = []struct {
	in       []Suggestion
	id       uint64
	out      *Suggestion
	outIndex int
}{
	{
		[]Suggestion{{ID: 1}, {ID: 2}},
		2,
		&Suggestion{ID: 2},
		1,
	},
	{
		[]Suggestion{},
		2,
		nil,
		-1,
	},
}

//
//sort.Sort(sort.Reverse(teian.ByUser(suggs)))
//sort.Sort(teian.ByDate(suggs))
//sort.Sort(sort.Reverse(teian.ByDate(suggs)))

func TestFindByID(t *testing.T) {
	for _, tt := range findByIDTests {
		sort.Sort(sort.Reverse(ByID(tt.in)))
		want, wantIndex := tt.out, tt.outIndex
		gotIndex, got := FindByID(tt.in, tt.id)
		if gotIndex != wantIndex {
			t.Errorf("FindByID(%v, %v) returned index %d, want %d", tt.in, tt.id, gotIndex, wantIndex)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("FindByID(%v, %v) returned suggestion \n%#v, want \n%#v", tt.in, tt.id, got, want)
		}
	}
}

var (
	today     = time.Now()
	yesterday = today.AddDate(0, 0, -1)
)

var filterByTextTests = []struct {
	in   []Suggestion
	text string
	out  []Suggestion
}{
	{
		[]Suggestion{{Text: "hello world", Created: yesterday}, {Text: "foo", Created: today}},
		"foo",
		[]Suggestion{{Text: "foo", Created: today}},
	},
	{
		[]Suggestion{},
		"foo",
		nil,
	},
}

func TestFilterByText(t *testing.T) {
	for _, tt := range filterByTextTests {
		sort.Sort(sort.Reverse(ByDate(tt.in)))
		if got, want := FilterByText(tt.in, tt.text), tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("FilterByText(%q, %q) = %q, want %q", tt.in, tt.text, got, want)
		}
	}
}

var filterByUserTests = []struct {
	in       []Suggestion
	username string
	out      []Suggestion
}{
	{
		[]Suggestion{{Username: "john"}, {Username: "mary"}},
		"john",
		[]Suggestion{{Username: "john"}},
	},
	{
		[]Suggestion{},
		"foo",
		nil,
	},
}

func TestFilterByUserText(t *testing.T) {
	for _, tt := range filterByUserTests {
		sort.Sort(sort.Reverse(ByUser(tt.in)))
		if got, want := FilterByUser(tt.in, tt.username), tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("FilterByUser(%q, %q) = %#v, want %#v", tt.in, tt.username, got, want)
		}
	}
}
