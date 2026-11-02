package cmd

import "testing"

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
