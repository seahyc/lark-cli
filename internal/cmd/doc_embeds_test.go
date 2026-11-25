package cmd

import (
	"reflect"
	"testing"

	"github.com/yjwong/lark-cli/internal/api"
)

func TestParseDocRef(t *testing.T) {
	cases := []struct {
		in   string
		want docRef
		err  bool
	}{
		{"ABC123", docRef{Token: "ABC123"}, false},
		{"https://glints.sg.larksuite.com/docx/LRRHdocdUoOyLmxrXm4lF9BhgSd?comment_id=7711910090546089733&comment_type=0&comment_anchor=true",
			docRef{Token: "LRRHdocdUoOyLmxrXm4lF9BhgSd", CommentID: "7711910090546089733"}, false},
		{"https://x.larksuite.com/docx/ABC", docRef{Token: "ABC"}, false},
		{"https://x.larksuite.com/wiki/WIKI1?from=x", docRef{Token: "WIKI1", Wiki: true}, false},
		{"https://x.larksuite.com/sheets/ABC", docRef{}, true},
	}
	for _, c := range cases {
		got, err := parseDocRef(c.in)
		if (err != nil) != c.err {
			t.Fatalf("%s: err=%v", c.in, err)
		}
		if !c.err && got != c.want {
			t.Fatalf("%s: got %+v want %+v", c.in, got, c.want)
		}
	}
}

func TestFlattenCellAndRef(t *testing.T) {
	v := []any{map[string]any{"type": "text", "text": "C4 The employer "}, map[string]any{"type": "text", "text": "tried"}}
	if got := flattenCell(v); got != "C4 The employer tried" {
		t.Fatal(got)
	}
	if flattenCell(float64(3)) != "3" || flattenCell(nil) != "" {
		t.Fatal("scalars")
	}
	if cellRef(3, 1) != "B4" || cellRef(0, 26) != "AA1" {
		t.Fatal("cellRef")
	}
}

func TestSheetGridAndFind(t *testing.T) {
	es := buildEmbeddedSheet("SS_sh", [][]any{
		{"h1", "h2", nil},
		{nil, []any{map[string]any{"text": "outside Bali"}}, nil},
		{nil, nil, nil},
	})
	if len(es.Grid) != 2 || len(es.Grid[0]) != 2 {
		t.Fatalf("grid %v", es.Grid)
	}
	m := es.find("outside bali")
	if len(m) != 1 || m[0].Cell != "B2" || m[0].SheetToken != "SS_sh" {
		t.Fatalf("%+v", m)
	}
	md := es.markdown()
	if want := "<!-- embedded sheet SS/sh -->\n| h1 | h2 |\n| --- | --- |\n| " + " | outside Bali |\n"; md != want {
		t.Fatalf("%q", md)
	}
}

func TestSelectAndSortComments(t *testing.T) {
	ip := func(i int) *int { return &i }
	cs := []api.OutputDocumentComment{
		{CommentID: "u"}, {CommentID: "c5", BlockIndex: ip(9)}, {CommentID: "c2", BlockIndex: ip(2)},
		{CommentID: "c3", BlockIndex: ip(4)}, {CommentID: "c4", BlockIndex: ip(6)},
	}
	sortCommentsDocOrder(cs)
	var ids []string
	for _, c := range cs {
		ids = append(ids, c.CommentID)
	}
	if !reflect.DeepEqual(ids, []string{"c2", "c3", "c4", "c5", "u"}) {
		t.Fatal(ids)
	}
	got := selectComments(cs, "", "c4", 1)
	if len(got) != 3 || got[0].CommentID != "c3" || !got[1].Target {
		t.Fatalf("%+v", got)
	}
	if one := selectComments(cs, "c2", "", 3); len(one) != 1 {
		t.Fatal("id")
	}
}
