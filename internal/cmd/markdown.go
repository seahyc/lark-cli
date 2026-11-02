package cmd

import (
	"bytes"
	"strings"

	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownTableData holds parsed table data from markdown for 2-phase table creation.
type MarkdownTableData struct {
	Headers     []string
	Rows        []string           // each row is pipe-separated cells
	HeaderCells []*api.TextBlock   // styled header cells (optional)
	RowCells    [][]*api.TextBlock // styled row cells (optional)
}

// pendingMarkdownTables collects tables found during markdown parsing.
// The caller (doc.go append) reads and drains this after parseMarkdownToBlocks returns.
var pendingMarkdownTables []MarkdownTableData

// parseMarkdownToBlocks converts markdown text into Lark document blocks.
func parseMarkdownToBlocks(source []byte) []api.DocumentBlock {
	pendingMarkdownTables = nil // reset
	md := goldmark.New(goldmark.WithExtensions(extension.Table))
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var blocks []api.DocumentBlock
	for node := doc.FirstChild(); node != nil; node = node.NextSibling() {
		blocks = append(blocks, convertNode(node, source)...)
	}
	return blocks
}

// convertNode converts an AST node to document blocks.
func convertNode(node ast.Node, source []byte) []api.DocumentBlock {
	switch n := node.(type) {
	case *ast.Heading:
		content := nodeText(n, source)
		level := n.Level
		if level < 1 {
			level = 1
		}
		if level > 9 {
			level = 9
		}
		block := api.DocumentBlock{BlockType: blockTypeForHeadingLevel(level)}
		tb := styledInline(n, source)
		if tb == nil {
			tb = makeTextBlock(content)
		}
		setHeadingField(&block, level, tb)
		return []api.DocumentBlock{block}

	case *ast.Paragraph:
		// Check if parent is a list item - handled by list case
		if _, ok := node.Parent().(*ast.ListItem); ok {
			return nil
		}
		tb := styledInline(n, source)
		if tb == nil {
			return nil
		}
		return []api.DocumentBlock{{BlockType: 2, Text: tb}}

	case *ast.FencedCodeBlock:
		var buf bytes.Buffer
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(source))
		}
		content := strings.TrimRight(buf.String(), "\n")
		tb := makeTextBlock(content)
		lang := string(n.Language(source))
		langID := markdownLangToID(lang)
		if langID > 0 {
			tb.Style = &api.TextStyle{Language: langID}
		}
		return []api.DocumentBlock{{BlockType: 14, Code: tb}}

	case *ast.CodeBlock:
		var buf bytes.Buffer
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(source))
		}
		content := strings.TrimRight(buf.String(), "\n")
		return []api.DocumentBlock{{BlockType: 14, Code: makeTextBlock(content)}}

	case *ast.List:
		var blocks []api.DocumentBlock
		blockType := 12 // bullet
		if n.IsOrdered() {
			blockType = 13 // ordered
		}
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if li, ok := child.(*ast.ListItem); ok {
				if tb := styledTextBlock(li, source); tb != nil {
					block := api.DocumentBlock{BlockType: blockType}
					if blockType == 12 {
						block.Bullet = tb
					} else {
						block.Ordered = tb
					}
					blocks = append(blocks, block)
				}
				// Nested lists become their own blocks after the parent item
				// instead of being glued onto the parent's text.
				for gc := li.FirstChild(); gc != nil; gc = gc.NextSibling() {
					if nested, ok := gc.(*ast.List); ok {
						blocks = append(blocks, convertNode(nested, source)...)
					}
				}
			}
		}
		return blocks

	case *ast.Blockquote:
		tb := styledTextBlock(n, source)
		if tb == nil {
			return nil
		}
		return []api.DocumentBlock{{BlockType: 15, Quote: tb}}

	case *ast.ThematicBreak:
		return []api.DocumentBlock{{BlockType: 22, Divider: &api.DividerBlock{}}}

	default:
		// Handle goldmark extension table nodes
		if table, ok := node.(*east.Table); ok {
			return convertMarkdownTable(table, source)
		}
		return nil
	}
}

// maxMarkdownTableCols caps columns per Lark table; wider markdown tables are
// split into adjacent sub-tables. Rows are not capped by Lark when tables are
// created through the descendant endpoint, but one request may carry at most
// maxDescendantBlocks blocks (table + cells + one paragraph per cell), so very
// long tables are split into consecutive chunks that repeat the header.
const (
	maxMarkdownTableCols = 9
	maxDescendantBlocks  = 1000
)

// rowsPerTableChunk returns how many data rows fit in one descendant request.
func rowsPerTableChunk(cols int) int {
	if cols <= 0 {
		return 1
	}
	n := (maxDescendantBlocks-1)/(2*cols) - 1 // minus the header row
	if n < 1 {
		n = 1
	}
	return n
}

// convertMarkdownTable extracts header and row data from a goldmark table AST
// node, stores it in pendingMarkdownTables, and returns one placeholder table
// block per (row chunk, column slice). Cell text keeps inline styles.
func convertMarkdownTable(table *east.Table, source []byte) []api.DocumentBlock {
	var headers []string
	var headerCells []*api.TextBlock
	var rows [][]string
	var rowCells [][]*api.TextBlock

	cellOf := func(cell ast.Node) (string, *api.TextBlock) {
		text := strings.TrimSpace(nodeText(cell, source))
		tb := styledInline(cell, source)
		if tb == nil {
			tb = makeTextBlock(text)
		}
		return text, tb
	}

	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		switch row := child.(type) {
		case *east.TableHeader:
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if _, ok := cell.(*east.TableCell); ok {
					t, tb := cellOf(cell)
					headers = append(headers, t)
					headerCells = append(headerCells, tb)
				}
			}
		case *east.TableRow:
			var texts []string
			var tbs []*api.TextBlock
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if _, ok := cell.(*east.TableCell); ok {
					t, tb := cellOf(cell)
					texts = append(texts, t)
					tbs = append(tbs, tb)
				}
			}
			rows = append(rows, texts)
			rowCells = append(rowCells, tbs)
		}
	}

	if len(headers) == 0 {
		return nil
	}

	colSlices := sliceColumns(len(headers), maxMarkdownTableCols)
	var out []api.DocumentBlock
	for _, cs := range colSlices {
		width := cs.end - cs.start
		chunk := rowsPerTableChunk(width)
		for r0 := 0; r0 == 0 || r0 < len(rows); r0 += chunk {
			r1 := r0 + chunk
			if r1 > len(rows) {
				r1 = len(rows)
			}
			data := MarkdownTableData{
				Headers:     append([]string(nil), headers[cs.start:cs.end]...),
				HeaderCells: append([]*api.TextBlock(nil), headerCells[cs.start:cs.end]...),
			}
			for i := r0; i < r1; i++ {
				texts := make([]string, width)
				tbs := make([]*api.TextBlock, width)
				for c := cs.start; c < cs.end; c++ {
					if c < len(rows[i]) {
						texts[c-cs.start] = rows[i][c]
						tbs[c-cs.start] = rowCells[i][c]
					}
				}
				data.Rows = append(data.Rows, strings.Join(texts, "|"))
				data.RowCells = append(data.RowCells, tbs)
			}
			pendingMarkdownTables = append(pendingMarkdownTables, data)
			out = append(out, api.DocumentBlock{
				BlockType: 31,
				Table: &api.TableBlock{
					Property: &api.TableProperty{
						RowSize:     1 + len(data.Rows),
						ColumnSize:  width,
						ColumnWidth: calcColumnWidths(data.Headers, data.Rows),
						HeaderRow:   true,
					},
				},
			})
			if len(rows) == 0 {
				break
			}
		}
	}
	return out
}

// columnSlice represents a half-open column index range [start, end).
type columnSlice struct{ start, end int }

// sliceColumns partitions [0, total) into contiguous slices each at most
// maxCols wide. total == 0 is treated as one empty slice so a header-only
// edge case still emits a placeholder.
func sliceColumns(total, maxCols int) []columnSlice {
	if maxCols <= 0 {
		maxCols = 1
	}
	if total <= 0 {
		return []columnSlice{{0, 0}}
	}
	var out []columnSlice
	for i := 0; i < total; i += maxCols {
		end := i + maxCols
		if end > total {
			end = total
		}
		out = append(out, columnSlice{i, end})
	}
	return out
}

// styledTextBlock converts a node's inline children into a TextBlock whose
// text runs keep markdown inline styling (bold, italic, inline code, links).
// Nested lists inside the node are skipped; callers emit them separately.
func styledTextBlock(node ast.Node, source []byte) *api.TextBlock {
	var elems []api.TextElement
	first := true
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, isList := child.(*ast.List); isList {
			continue
		}
		var part []api.TextElement
		switch child.(type) {
		case *ast.Paragraph, *ast.TextBlock:
			part = inlineElements(kidsOf(child), source, api.TextElementStyle{})
			if !first && len(part) > 0 {
				elems = appendRun(elems, "\n", api.TextElementStyle{})
			}
		default:
			part = inlineElements([]ast.Node{child}, source, api.TextElementStyle{})
		}
		for _, e := range part {
			elems = appendRun(elems, e.TextRun.Content, styleValue(e.TextRun.TextElementStyle))
		}
		if len(part) > 0 {
			first = false
		}
	}
	// Trim trailing newlines left by soft/hard breaks.
	for len(elems) > 0 {
		last := elems[len(elems)-1].TextRun
		trimmed := strings.TrimRight(last.Content, "\n")
		if trimmed != "" {
			last.Content = trimmed
			break
		}
		elems = elems[:len(elems)-1]
	}
	if len(elems) == 0 {
		return nil
	}
	return &api.TextBlock{Elements: elems}
}

// styledInline converts an inline container (heading, paragraph) into a
// styled TextBlock.
func styledInline(n ast.Node, source []byte) *api.TextBlock {
	elems := inlineElements(kidsOf(n), source, api.TextElementStyle{})
	for len(elems) > 0 {
		last := elems[len(elems)-1].TextRun
		trimmed := strings.TrimRight(last.Content, "\n")
		if trimmed != "" {
			last.Content = trimmed
			break
		}
		elems = elems[:len(elems)-1]
	}
	if len(elems) == 0 {
		return nil
	}
	return &api.TextBlock{Elements: elems}
}

// kidsOf returns a node's direct children.
func kidsOf(n ast.Node) []ast.Node {
	var kids []ast.Node
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		kids = append(kids, c)
	}
	return kids
}

// inlineElements walks inline nodes and returns styled text runs.
func inlineElements(kids []ast.Node, source []byte, style api.TextElementStyle) []api.TextElement {
	var out []api.TextElement
	for _, child := range kids {
		switch c := child.(type) {
		case *ast.Text:
			content := string(c.Segment.Value(source))
			if c.HardLineBreak() || c.SoftLineBreak() {
				content += "\n"
			}
			out = appendRun(out, content, style)
		case *ast.String:
			out = appendRun(out, string(c.Value), style)
		case *ast.CodeSpan:
			s := style
			s.InlineCode = true
			var buf bytes.Buffer
			for gc := c.FirstChild(); gc != nil; gc = gc.NextSibling() {
				if t, ok := gc.(*ast.Text); ok {
					buf.Write(t.Segment.Value(source))
				}
			}
			out = appendRun(out, buf.String(), s)
		case *ast.Emphasis:
			s := style
			if c.Level >= 2 {
				s.Bold = true
			} else {
				s.Italic = true
			}
			for _, e := range inlineElements(kidsOf(c), source, s) {
				out = appendRun(out, e.TextRun.Content, styleValue(e.TextRun.TextElementStyle))
			}
		case *ast.Link:
			s := style
			s.Link = &api.TextLink{URL: string(c.Destination)}
			for _, e := range inlineElements(kidsOf(c), source, s) {
				out = appendRun(out, e.TextRun.Content, styleValue(e.TextRun.TextElementStyle))
			}
		case *ast.AutoLink:
			s := style
			url := string(c.URL(source))
			s.Link = &api.TextLink{URL: url}
			out = appendRun(out, url, s)
		case *ast.List:
			// handled by the list converter
		default:
			if c.HasChildren() {
				for _, e := range inlineElements(kidsOf(c), source, style) {
					out = appendRun(out, e.TextRun.Content, styleValue(e.TextRun.TextElementStyle))
				}
			}
		}
	}
	return out
}

func styleValue(s *api.TextElementStyle) api.TextElementStyle {
	if s == nil {
		return api.TextElementStyle{}
	}
	return *s
}

// appendRun appends content as a text run, merging with the previous run when
// the styles match.
func appendRun(elems []api.TextElement, content string, style api.TextElementStyle) []api.TextElement {
	if content == "" {
		return elems
	}
	if n := len(elems); n > 0 && sameStyle(styleValue(elems[n-1].TextRun.TextElementStyle), style) {
		elems[n-1].TextRun.Content += content
		return elems
	}
	run := &api.TextRun{Content: content}
	if !sameStyle(style, api.TextElementStyle{}) {
		s := style
		run.TextElementStyle = &s
	}
	return append(elems, api.TextElement{TextRun: run})
}

func sameStyle(a, b api.TextElementStyle) bool {
	la, lb := "", ""
	if a.Link != nil {
		la = a.Link.URL
	}
	if b.Link != nil {
		lb = b.Link.URL
	}
	a.Link, b.Link = nil, nil
	return a == b && la == lb
}

// nodeText extracts text content from an AST node.
func nodeText(node ast.Node, source []byte) string {
	var buf bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch c := child.(type) {
		case *ast.Text:
			buf.Write(c.Segment.Value(source))
			if c.HardLineBreak() || c.SoftLineBreak() {
				buf.WriteByte('\n')
			}
		case *ast.CodeSpan:
			for gc := c.FirstChild(); gc != nil; gc = gc.NextSibling() {
				if t, ok := gc.(*ast.Text); ok {
					buf.Write(t.Segment.Value(source))
				}
			}
		case *ast.Emphasis:
			buf.WriteString(nodeText(c, source))
		case *ast.Link:
			buf.WriteString(nodeText(c, source))
		case *ast.AutoLink:
			buf.Write(c.URL(source))
		default:
			if c.HasChildren() {
				buf.WriteString(nodeText(c, source))
			}
		}
	}
	return buf.String()
}

// markdownLangToID maps common markdown language names to Lark code language IDs.
func markdownLangToID(lang string) int {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "bash", "sh", "shell":
		return 7
	case "c":
		return 9
	case "cpp", "c++":
		return 10
	case "csharp", "c#", "cs":
		return 12
	case "css":
		return 13
	case "go", "golang":
		return 22
	case "html":
		return 25
	case "java":
		return 29
	case "javascript", "js":
		return 30
	case "json":
		return 31
	case "kotlin":
		return 32
	case "markdown", "md":
		return 35
	case "php":
		return 42
	case "python", "py":
		return 49
	case "ruby", "rb":
		return 51
	case "rust", "rs":
		return 53
	case "scala":
		return 54
	case "sql":
		return 56
	case "swift":
		return 58
	case "typescript", "ts":
		return 63
	case "xml":
		return 66
	case "yaml", "yml":
		return 67
	default:
		return 1 // PlainText
	}
}
