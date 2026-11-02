package cmd

import (
	"testing"

	"github.com/yjwong/lark-cli/internal/api"
)

func TestParseMarkdownInlineStyles(t *testing.T) {
	blocks := parseMarkdownToBlocks([]byte("Plain **bold** and *ital* and `code` and [link](https://x.test).\n"))
	if len(blocks) != 1 || blocks[0].Text == nil {
		t.Fatalf("want one text block, got %+v", blocks)
	}
	els := blocks[0].Text.Elements
	got := map[string]string{}
	for _, e := range els {
		s := e.TextRun.TextElementStyle
		switch {
		case s == nil:
			got[e.TextRun.Content] = "plain"
		case s.Bold:
			got[e.TextRun.Content] = "bold"
		case s.Italic:
			got[e.TextRun.Content] = "italic"
		case s.InlineCode:
			got[e.TextRun.Content] = "code"
		case s.Link != nil:
			got[e.TextRun.Content] = "link:" + s.Link.URL
		}
	}
	want := map[string]string{"bold": "bold", "ital": "italic", "code": "code", "link": "link:https://x.test"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("run %q: got %q want %q (all: %v)", k, got[k], v, got)
		}
	}
}

func TestParseMarkdownNestedListsAreSeparateBlocks(t *testing.T) {
	blocks := parseMarkdownToBlocks([]byte("- parent **b**\n  - child one\n  - child two\n- next\n"))
	var texts []string
	for _, b := range blocks {
		if b.Bullet == nil {
			t.Fatalf("expected only bullets, got %+v", b)
		}
		s := ""
		for _, e := range b.Bullet.Elements {
			s += e.TextRun.Content
		}
		texts = append(texts, s)
	}
	want := []string{"parent b", "child one", "child two", "next"}
	if len(texts) != len(want) {
		t.Fatalf("got %q want %q", texts, want)
	}
	for i := range want {
		if texts[i] != want[i] {
			t.Errorf("block %d: got %q want %q", i, texts[i], want[i])
		}
	}
	if !blocks[0].Bullet.Elements[1].TextRun.TextElementStyle.Bold {
		t.Errorf("bold run inside bullet lost its style")
	}
}

func TestMarkdownTableKeepsRowsStylesAndFitsPage(t *testing.T) {
	src := "| Product | Price |\n|---|---|\n"
	for i := 0; i < 12; i++ {
		src += "| **VIP** item | IDR 360K (US$23) |\n"
	}
	blocks := parseMarkdownToBlocks([]byte(src))
	if len(blocks) != 1 || blocks[0].BlockType != 31 {
		t.Fatalf("want a single table placeholder, got %d blocks", len(blocks))
	}
	if got := blocks[0].Table.Property.RowSize; got != 13 {
		t.Errorf("row size = %d, want 13 (header + 12 rows in one table)", got)
	}
	sum := 0
	for _, w := range blocks[0].Table.Property.ColumnWidth {
		sum += w
	}
	if sum > tablePageWidth {
		t.Errorf("column widths sum to %d, wider than the page (%d)", sum, tablePageWidth)
	}
	if len(pendingMarkdownTables) != 1 {
		t.Fatalf("pending tables = %d, want 1", len(pendingMarkdownTables))
	}
	cell := pendingMarkdownTables[0].RowCells[0][0]
	if cell == nil || len(cell.Elements) == 0 || cell.Elements[0].TextRun.TextElementStyle == nil || !cell.Elements[0].TextRun.TextElementStyle.Bold {
		t.Errorf("bold markdown in a table cell lost its style: %+v", cell)
	}
}

func TestNonEmptyCellReplacesEmptyText(t *testing.T) {
	for _, tb := range []*api.TextBlock{nil, makeTextBlock(""), {}} {
		got := nonEmptyCell(tb)
		if len(got.Elements) != 1 || got.Elements[0].TextRun.Content != " " {
			t.Errorf("nonEmptyCell(%+v) = %+v, want a single-space run", tb, got)
		}
	}
	keep := makeTextBlock("x")
	if nonEmptyCell(keep) != keep {
		t.Errorf("non-empty cell was replaced")
	}
}
