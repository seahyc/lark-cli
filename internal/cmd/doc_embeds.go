package cmd

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/yjwong/lark-cli/internal/api"
)

const (
	embedMaxRows   = 200
	embedMaxCols   = 26
	blockTypeSheet = 30
)

// --- doc reference parsing ---

// docRef is a parsed document argument: a bare token or a Lark URL.
type docRef struct {
	Token     string
	Wiki      bool   // Token is a wiki node token that still needs resolving
	CommentID string // comment_id query parameter, if any
}

var docURLPath = regexp.MustCompile(`^/(docx|docs|wiki)/([A-Za-z0-9]+)/?$`)

// parseDocRef accepts a bare token or a https://<host>/docx/<token>?... or
// /wiki/<token> URL.
func parseDocRef(arg string) (docRef, error) {
	arg = strings.TrimSpace(arg)
	if !strings.Contains(arg, "://") {
		return docRef{Token: arg}, nil
	}
	u, err := url.Parse(arg)
	if err != nil {
		return docRef{}, fmt.Errorf("invalid document URL %q: %w", arg, err)
	}
	m := docURLPath.FindStringSubmatch(u.Path)
	if m == nil {
		return docRef{}, fmt.Errorf("unsupported document URL %q (expected /docx/<token> or /wiki/<token>)", arg)
	}
	return docRef{Token: m[2], Wiki: m[1] == "wiki", CommentID: u.Query().Get("comment_id")}, nil
}

// resolveDocArg turns a doc argument into a docx document ID (resolving wiki
// nodes) and returns any comment_id carried by the URL.
func resolveDocArg(client *api.Client, arg string) (string, string, error) {
	ref, err := parseDocRef(arg)
	if err != nil {
		return "", "", err
	}
	if !ref.Wiki {
		return ref.Token, ref.CommentID, nil
	}
	node, err := client.GetWikiNode(ref.Token)
	if err != nil {
		return "", "", fmt.Errorf("resolve wiki node %s: %w", ref.Token, err)
	}
	if node.ObjType != "docx" {
		return "", "", fmt.Errorf("wiki node %s is a %q, not a docx document", ref.Token, node.ObjType)
	}
	return node.ObjToken, ref.CommentID, nil
}

// --- block ordering ---

// blocksInDocOrder returns blocks in depth-first document order.
func blocksInDocOrder(blocks []api.DocumentBlock) []api.DocumentBlock {
	byID := make(map[string]api.DocumentBlock, len(blocks))
	for _, b := range blocks {
		byID[b.BlockID] = b
	}
	var root string
	for _, b := range blocks {
		if b.ParentID == "" {
			root = b.BlockID
			break
		}
	}
	if root == "" {
		return blocks
	}
	out := make([]api.DocumentBlock, 0, len(blocks))
	seen := map[string]bool{}
	var walk func(id string)
	walk = func(id string) {
		b, ok := byID[id]
		if !ok || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, b)
		for _, c := range b.Children {
			walk(c)
		}
	}
	walk(root)
	for _, b := range blocks { // orphans
		if !seen[b.BlockID] {
			out = append(out, b)
		}
	}
	return out
}

// splitSheetToken splits "<spreadsheetToken>_<sheetId>".
func splitSheetToken(tok string) (string, string, bool) {
	i := strings.LastIndex(tok, "_")
	if i <= 0 || i == len(tok)-1 {
		return "", "", false
	}
	return tok[:i], tok[i+1:], true
}

// --- sheet cell handling ---

// flattenCell renders a sheet cell value (string, number, bool, rich-text
// segment list, or link object) as plain text.
func flattenCell(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case []any:
		var sb strings.Builder
		for _, seg := range x {
			sb.WriteString(flattenCell(seg))
		}
		return sb.String()
	case map[string]any:
		for _, k := range []string{"text", "link", "name"} {
			if s, ok := x[k]; ok {
				if str := flattenCell(s); str != "" {
					return str
				}
			}
		}
		return ""
	default:
		return fmt.Sprint(x)
	}
}

// cellRef returns the A1 reference for zero-based row/col.
func cellRef(row, col int) string {
	name := ""
	for c := col + 1; c > 0; c = (c - 1) / 26 {
		name = string(rune('A'+(c-1)%26)) + name
	}
	return name + strconv.Itoa(row+1)
}

// embeddedSheet holds the flattened text grid of an embedded sheet.
type embeddedSheet struct {
	Token     string // "<spreadsheetToken>_<sheetId>"
	Grid      [][]string
	Truncated bool
}

// fetchEmbeddedSheet loads and flattens the used range of an embedded sheet.
func fetchEmbeddedSheet(client *api.Client, sheetToken string) (*embeddedSheet, error) {
	ss, sheetID, ok := splitSheetToken(sheetToken)
	if !ok {
		return nil, fmt.Errorf("unrecognised sheet token %q", sheetToken)
	}
	rng := fmt.Sprintf("%s!A1:%s%d", sheetID, cellRef(0, embedMaxCols-1)[:1], embedMaxRows+1)
	data, err := client.GetSheetData(ss, rng, "")
	if err != nil {
		return nil, err
	}
	var raw [][]any
	if data != nil && data.ValueRange != nil {
		raw = data.ValueRange.Values
	}
	return buildEmbeddedSheet(sheetToken, raw), nil
}

// buildEmbeddedSheet flattens raw values and trims empty trailing rows/cols,
// capping at embedMaxRows x embedMaxCols (Truncated is set if data was cut).
func buildEmbeddedSheet(token string, raw [][]any) *embeddedSheet {
	es := &embeddedSheet{Token: token}
	maxCol := 0
	for r, row := range raw {
		if r >= embedMaxRows {
			es.Truncated = true
			break
		}
		cells := make([]string, len(row))
		for c, v := range row {
			cells[c] = flattenCell(v)
		}
		if len(cells) > embedMaxCols {
			cells = cells[:embedMaxCols]
			es.Truncated = true
		}
		es.Grid = append(es.Grid, cells)
	}
	for _, row := range es.Grid {
		for c, v := range row {
			if strings.TrimSpace(v) != "" && c+1 > maxCol {
				maxCol = c + 1
			}
		}
	}
	for len(es.Grid) > 0 {
		last := es.Grid[len(es.Grid)-1]
		empty := true
		for _, v := range last {
			if strings.TrimSpace(v) != "" {
				empty = false
				break
			}
		}
		if !empty {
			break
		}
		es.Grid = es.Grid[:len(es.Grid)-1]
	}
	for i, row := range es.Grid {
		for len(row) < maxCol {
			row = append(row, "")
		}
		es.Grid[i] = row[:maxCol]
	}
	return es
}

func escapeMDCell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\r", "")
	return strings.ReplaceAll(s, "\n", "<br>")
}

// markdown renders the sheet as a heading comment plus a markdown table.
func (es *embeddedSheet) markdown() string {
	var sb strings.Builder
	ss, id := mustSplit(es.Token)
	sb.WriteString(fmt.Sprintf("<!-- embedded sheet %s/%s -->\n", ss, id))
	if len(es.Grid) == 0 || len(es.Grid[0]) == 0 {
		sb.WriteString("(empty sheet)\n")
		return sb.String()
	}
	cols := len(es.Grid[0])
	for i, row := range es.Grid {
		cells := make([]string, cols)
		for c := range cells {
			cells[c] = escapeMDCell(row[c])
		}
		sb.WriteString("| " + strings.Join(cells, " | ") + " |\n")
		if i == 0 {
			sep := make([]string, cols)
			for c := range sep {
				sep[c] = "---"
			}
			sb.WriteString("| " + strings.Join(sep, " | ") + " |\n")
		}
	}
	if es.Truncated {
		sb.WriteString(fmt.Sprintf("\n_(truncated to %d rows x %d cols)_\n", embedMaxRows, embedMaxCols))
	}
	return sb.String()
}

func mustSplit(tok string) (string, string) {
	a, b, ok := splitSheetToken(tok)
	if !ok {
		return tok, ""
	}
	return a, b
}

// embeddedSheetsMarkdown fetches all embedded sheets in doc order and renders
// them. A failed fetch becomes a one-line placeholder.
func embeddedSheetsMarkdown(client *api.Client, blocks []api.DocumentBlock) string {
	var sb strings.Builder
	for _, b := range blocksInDocOrder(blocks) {
		if b.BlockType != blockTypeSheet || b.Sheet == nil || b.Sheet.Token == "" {
			continue
		}
		es, err := fetchEmbeddedSheet(client, b.Sheet.Token)
		if err != nil {
			ss, id := mustSplit(b.Sheet.Token)
			sb.WriteString(fmt.Sprintf("\n<!-- embedded sheet %s/%s (block %s): could not load: %v -->\n", ss, id, b.BlockID, err))
			continue
		}
		sb.WriteString(fmt.Sprintf("\n<!-- block %s -->\n", b.BlockID))
		sb.WriteString(es.markdown())
	}
	return sb.String()
}

// findInGrid returns cell matches (case-insensitive substring).
func (es *embeddedSheet) find(queryLower string) []api.OutputFindMatch {
	var out []api.OutputFindMatch
	for r, row := range es.Grid {
		for c, v := range row {
			if v == "" || !strings.Contains(strings.ToLower(v), queryLower) {
				continue
			}
			preview := v
			if len(preview) > 120 {
				preview = preview[:120] + "..."
			}
			out = append(out, api.OutputFindMatch{
				BlockType:  blockTypeSheet,
				Preview:    preview,
				SheetToken: es.Token,
				Cell:       cellRef(r, c),
			})
		}
	}
	return out
}
