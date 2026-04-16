package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
)

// Format controls how JSON() renders values. Set via the global --format flag
// on the root command. Defaults to "pretty".
var Format = "pretty"

const (
	FormatPretty = "pretty" // indented JSON (legacy default)
	FormatJSON   = "json"   // alias for pretty
	FormatNDJSON = "ndjson" // compact one-line JSON
	FormatTable  = "table"  // ASCII table for arrays/object-of-arrays
	FormatCSV    = "csv"    // CSV for arrays
)

// ValidFormats returns the supported --format values.
func ValidFormats() []string {
	return []string{FormatPretty, FormatJSON, FormatNDJSON, FormatTable, FormatCSV}
}

// JSON outputs data to stdout in the currently configured Format.
func JSON(v interface{}) {
	switch Format {
	case FormatNDJSON:
		enc := json.NewEncoder(os.Stdout)
		_ = enc.Encode(v) // Go's json.Encoder compacts by default + writes trailing newline
	case FormatTable:
		renderTable(os.Stdout, v)
	case FormatCSV:
		renderCSV(os.Stdout, v)
	default: // pretty / json
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(v)
	}
}

// Error outputs an error in JSON format — always pretty-printed regardless of
// Format, so tooling that inspects stderr gets a stable shape.
func Error(code, message string) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]interface{}{
		"error":   true,
		"code":    code,
		"message": message,
	})
}

// ErrorFromErr outputs an error from a Go error
func ErrorFromErr(code string, err error) {
	Error(code, err.Error())
}

// Success outputs a success message
func Success(message string) {
	JSON(map[string]interface{}{
		"success": true,
		"message": message,
	})
}

// Fatal outputs an error and exits with code 1
func Fatal(code string, err error) {
	Error(code, err.Error())
	os.Exit(1)
}

// Fatalf outputs a formatted error and exits
func Fatalf(code, format string, args ...interface{}) {
	Error(code, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// --- Table / CSV rendering ---

// extractRows finds the slice-of-objects inside v and returns rows + column order.
// If v is already a slice, it's used directly. If v is a map with exactly one
// slice value (e.g. {"tasks": [...], "count": 3}), that slice is used. Otherwise,
// v is treated as a single row.
func extractRows(v interface{}) ([]map[string]interface{}, []string) {
	rows := toRows(v)
	if rows == nil {
		// Fallback: treat as single object
		single := toMap(v)
		if single == nil {
			return nil, nil
		}
		rows = []map[string]interface{}{single}
	}

	// Collect columns in insertion-ish order: first appearance wins.
	seen := map[string]bool{}
	var cols []string
	for _, r := range rows {
		// Sort keys per-row for determinism, then add unseen ones.
		keys := make([]string, 0, len(r))
		for k := range r {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !seen[k] {
				seen[k] = true
				cols = append(cols, k)
			}
		}
	}
	return rows, cols
}

func toRows(v interface{}) []map[string]interface{} {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return sliceToRows(rv)
	case reflect.Map, reflect.Struct:
		// Round-trip through JSON to get a uniform map, then look for the
		// sole slice-valued field.
		m := toMap(v)
		if m == nil {
			return nil
		}
		var sliceKey string
		sliceCount := 0
		for k, val := range m {
			if _, ok := val.([]interface{}); ok {
				sliceKey = k
				sliceCount++
			}
		}
		if sliceCount == 1 {
			arr, _ := m[sliceKey].([]interface{})
			return interfaceSliceToRows(arr)
		}
	}
	return nil
}

func sliceToRows(rv reflect.Value) []map[string]interface{} {
	arr := make([]interface{}, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		arr[i] = rv.Index(i).Interface()
	}
	return interfaceSliceToRows(arr)
}

func interfaceSliceToRows(arr []interface{}) []map[string]interface{} {
	if len(arr) == 0 {
		return []map[string]interface{}{}
	}
	rows := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		m := toMap(item)
		if m == nil {
			// Wrap scalars as {"value": x}
			m = map[string]interface{}{"value": item}
		}
		rows = append(rows, m)
	}
	return rows
}

func toMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	// Round-trip via JSON for structs / other maps.
	bs, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(bs, &m); err != nil {
		return nil
	}
	return m
}

func cellString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case bool, float64, float32, int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%v", x)
	default:
		bs, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(bs)
	}
}

func renderTable(w *os.File, v interface{}) {
	rows, cols := extractRows(v)
	if rows == nil || len(cols) == 0 {
		// Fall back to pretty JSON for things that don't fit the table shape.
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(v)
		return
	}

	// Compute column widths.
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c)
	}
	strRows := make([][]string, len(rows))
	for i, r := range rows {
		strRows[i] = make([]string, len(cols))
		for j, c := range cols {
			s := cellString(r[c])
			if len(s) > 80 {
				s = s[:77] + "..."
			}
			strRows[i][j] = s
			if len(s) > widths[j] {
				widths[j] = len(s)
			}
		}
	}

	sep := separator(widths)
	fmt.Fprintln(w, sep)
	fmt.Fprintln(w, rowLine(cols, widths))
	fmt.Fprintln(w, sep)
	for _, r := range strRows {
		fmt.Fprintln(w, rowLine(r, widths))
	}
	fmt.Fprintln(w, sep)
}

func rowLine(cells []string, widths []int) string {
	var sb strings.Builder
	sb.WriteString("|")
	for i, c := range cells {
		sb.WriteString(" ")
		sb.WriteString(c)
		pad := widths[i] - len(c)
		if pad > 0 {
			sb.WriteString(strings.Repeat(" ", pad))
		}
		sb.WriteString(" |")
	}
	return sb.String()
}

func separator(widths []int) string {
	var sb strings.Builder
	sb.WriteString("+")
	for _, w := range widths {
		sb.WriteString(strings.Repeat("-", w+2))
		sb.WriteString("+")
	}
	return sb.String()
}

func renderCSV(w *os.File, v interface{}) {
	rows, cols := extractRows(v)
	if rows == nil || len(cols) == 0 {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(v)
		return
	}
	cw := csv.NewWriter(w)
	_ = cw.Write(cols)
	for _, r := range rows {
		line := make([]string, len(cols))
		for i, c := range cols {
			line[i] = cellString(r[c])
		}
		_ = cw.Write(line)
	}
	cw.Flush()
}
