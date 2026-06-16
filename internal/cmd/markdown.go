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
	Headers []string
	Rows    []string // each row is pipe-separated cells
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
		setHeadingField(&block, level, makeTextBlock(content))
		return []api.DocumentBlock{block}

	case *ast.Paragraph:
		// Check if parent is a list item - handled by list case
		if _, ok := node.Parent().(*ast.ListItem); ok {
			return nil
		}
		content := nodeText(n, source)
		if content == "" {
			return nil
		}
		return []api.DocumentBlock{{BlockType: 2, Text: makeTextBlock(content)}}

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
				content := nodeText(li, source)
				tb := makeTextBlock(content)
				block := api.DocumentBlock{BlockType: blockType}
				if blockType == 12 {
					block.Bullet = tb
				} else {
					block.Ordered = tb
				}
				blocks = append(blocks, block)
			}
		}
		return blocks

	case *ast.Blockquote:
		var parts []string
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			t := nodeText(child, source)
			if t != "" {
				parts = append(parts, t)
			}
		}
		content := strings.Join(parts, "\n")
		return []api.DocumentBlock{{BlockType: 15, Quote: makeTextBlock(content)}}

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

// maxMarkdownTableRows / maxMarkdownTableCols cap how many data rows and
// columns can go into a single Lark table when importing from markdown.
// Empirically Lark's docx create-blocks endpoint rejects any table whose
// row_size or column_size exceeds 9 with code 1770001 ("invalid param");
// we use 8 for rows (matching the previous 6+header convention) and 9 for
// columns. Tables exceeding either limit are split into a grid of adjacent
// sub-tables that render acceptably in the Lark UI.
const (
	maxMarkdownTableRows = 8
	maxMarkdownTableCols = 9
)

// convertMarkdownTable extracts header and row data from a goldmark table AST
// node, stores it in pendingMarkdownTables for 2-phase creation, and returns
// one or more placeholder table blocks. Tables with more than
// maxMarkdownTableRows data rows are split into chunks; each chunk repeats the
// header so the visual result matches a single big table.
func convertMarkdownTable(table *east.Table, source []byte) []api.DocumentBlock {
	var headers []string
	var rows []string

	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		switch row := child.(type) {
		case *east.TableHeader:
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if _, ok := cell.(*east.TableCell); ok {
					headers = append(headers, strings.TrimSpace(nodeText(cell, source)))
				}
			}
		case *east.TableRow:
			var cells []string
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if _, ok := cell.(*east.TableCell); ok {
					cells = append(cells, strings.TrimSpace(nodeText(cell, source)))
				}
			}
			rows = append(rows, strings.Join(cells, "|"))
		}
	}

	if len(headers) == 0 {
		return nil
	}

	// First chunk vertically (by rows), then within each row-chunk slice
	// horizontally (by columns). Each (rowChunk, colChunk) pair becomes one
	// pendingMarkdownTable + placeholder block. Order: top-to-bottom, then
	// left-to-right within a row-chunk, so the visual reading order matches
	// the source markdown when blocks are rendered sequentially.
	rowChunks := chunkTableRows(rows, maxMarkdownTableRows)
	colSlices := sliceColumns(len(headers), maxMarkdownTableCols)

	out := make([]api.DocumentBlock, 0, len(rowChunks)*len(colSlices))
	for _, rowChunk := range rowChunks {
		for _, cs := range colSlices {
			subHeaders := headers[cs.start:cs.end]
			subRows := projectRowColumns(rowChunk, cs.start, cs.end, len(headers))
			pendingMarkdownTables = append(pendingMarkdownTables, MarkdownTableData{
				Headers: subHeaders,
				Rows:    subRows,
			})
			colSize := len(subHeaders)
			rowSize := 1 + len(subRows)
			out = append(out, api.DocumentBlock{
				BlockType: 31,
				Table: &api.TableBlock{
					Property: &api.TableProperty{
						RowSize:     rowSize,
						ColumnSize:  colSize,
						ColumnWidth: calcColumnWidths(subHeaders, subRows),
						HeaderRow:   true,
					},
				},
			})
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

// projectRowColumns extracts columns [colStart, colEnd) from a slice of rows
// encoded as "c1|c2|..." strings. Missing cells are emitted as empty.
func projectRowColumns(rows []string, colStart, colEnd, totalCols int) []string {
	if rows == nil {
		return nil
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		cells := strings.SplitN(row, "|", totalCols+1)
		sub := make([]string, 0, colEnd-colStart)
		for i := colStart; i < colEnd; i++ {
			if i < len(cells) {
				sub = append(sub, strings.TrimSpace(cells[i]))
			} else {
				sub = append(sub, "")
			}
		}
		out = append(out, strings.Join(sub, "|"))
	}
	return out
}

// chunkTableRows splits a row slice into chunks of at most maxRows entries.
// An empty input yields a single empty chunk so the header-only table is
// still emitted.
func chunkTableRows(rows []string, maxRows int) [][]string {
	if maxRows <= 0 {
		maxRows = 1
	}
	if len(rows) == 0 {
		return [][]string{nil}
	}
	var chunks [][]string
	for i := 0; i < len(rows); i += maxRows {
		end := i + maxRows
		if end > len(rows) {
			end = len(rows)
		}
		chunks = append(chunks, rows[i:end])
	}
	return chunks
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
