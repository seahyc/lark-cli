package api

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// llmsModules maps short module names to their llms-docs URLs on open.larksuite.com.
// These files hold the entire module's API reference in plain text form suitable
// for grep-and-extract lookups.
var llmsModules = map[string]string{
	"contacts":   "https://open.larksuite.com/llms-docs/en-US/llms-contacts.txt",
	"messaging":  "https://open.larksuite.com/llms-docs/en-US/llms-messaging.txt",
	"messages":   "https://open.larksuite.com/llms-docs/en-US/llms-messaging.txt", // alias
	"im":         "https://open.larksuite.com/llms-docs/en-US/llms-messaging.txt", // alias
	"group-chat": "https://open.larksuite.com/llms-docs/en-US/llms-group-chat.txt",
	"feed":       "https://open.larksuite.com/llms-docs/en-US/llms-feed.txt",
	"docs":       "https://open.larksuite.com/llms-docs/en-US/llms-docs.txt",
	"calendar":   "https://open.larksuite.com/llms-docs/en-US/llms-calendar.txt",
	"meetings":   "https://open.larksuite.com/llms-docs/en-US/llms-video-conferencing.txt",
	"vc":         "https://open.larksuite.com/llms-docs/en-US/llms-video-conferencing.txt",
	"attendance": "https://open.larksuite.com/llms-docs/en-US/llms-attendance.txt",
	"approval":   "https://open.larksuite.com/llms-docs/en-US/llms-approval.txt",
	"bot":        "https://open.larksuite.com/llms-docs/en-US/llms-bot.txt",
	"tasks":      "https://open.larksuite.com/llms-docs/en-US/llms-tasks-v2.txt",
	"mail":       "https://open.larksuite.com/llms-docs/en-US/llms-email.txt",
	"email":      "https://open.larksuite.com/llms-docs/en-US/llms-email.txt",
	"bitable":    "https://open.larksuite.com/llms-docs/en-US/llms-docs.txt", // bitable lives inside docs
}

// InferModule guesses the llms-docs module name from an API path or dotted name.
// Examples:
//   - "/open-apis/im/v1/messages"     → "im"
//   - "/im/v1/messages"               → "im"
//   - "im.messages.send"              → "im"
//   - "approval/v4/instances/query"   → "approval"
func InferModule(pathOrName string) string {
	s := strings.TrimSpace(pathOrName)
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimPrefix(s, "open-apis/")

	// Dotted form: "im.messages.send"
	if strings.Contains(s, ".") && !strings.Contains(s, "/") {
		return strings.SplitN(s, ".", 2)[0]
	}
	// Path form: "im/v1/messages"
	if parts := strings.SplitN(s, "/", 2); len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// FetchSchemaSection fetches the llms-docs for the inferred module and returns
// the text section(s) that reference the given path or name. The returned
// sections are the consecutive lines around each match, delimited by the
// llms-docs's top-level "## " headings.
func FetchSchemaSection(pathOrName string) (module, url, section string, err error) {
	module = InferModule(pathOrName)
	if module == "" {
		return "", "", "", fmt.Errorf("could not infer module from %q", pathOrName)
	}
	url, ok := llmsModules[module]
	if !ok {
		return module, "", "", fmt.Errorf("no llms-docs mapping for module %q (known: %v)", module, knownModules())
	}

	body, err := fetchText(url)
	if err != nil {
		return module, url, "", err
	}

	// Normalize the search needle: for path form we want to match the canonical
	// "/open-apis/..." appearance; dotted form we match loosely on the tail.
	needles := searchNeedles(pathOrName)

	section = extractSections(body, needles)
	if section == "" {
		return module, url, "", fmt.Errorf("no matching section found for %q in %s llms-docs", pathOrName, module)
	}
	return module, url, section, nil
}

func knownModules() []string {
	ks := make([]string, 0, len(llmsModules))
	for k := range llmsModules {
		ks = append(ks, k)
	}
	return ks
}

// searchNeedles produces alternative substrings to match against doc text.
func searchNeedles(pathOrName string) []string {
	s := strings.TrimSpace(pathOrName)
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimPrefix(s, "open-apis/")

	var out []string
	if strings.Contains(s, "/") {
		out = append(out, "/open-apis/"+s, "/"+s)
	} else if strings.Contains(s, ".") {
		// "im.messages.send" → match "messages/send" or "send"
		parts := strings.Split(s, ".")
		tail := strings.Join(parts[1:], "/")
		if tail != "" {
			out = append(out, tail)
		}
		out = append(out, parts[len(parts)-1])
	} else {
		out = append(out, s)
	}
	return out
}

// extractSections finds every ## or ### heading in body whose section body
// contains at least one needle, and returns their contents concatenated.
func extractSections(body string, needles []string) string {
	sectionStart := regexp.MustCompile(`(?m)^##\s+`)

	// Find all ## heading offsets
	indices := sectionStart.FindAllStringIndex(body, -1)
	if len(indices) == 0 {
		// No sectioning — dump the whole thing if any needle matches.
		for _, n := range needles {
			if strings.Contains(body, n) {
				return body
			}
		}
		return ""
	}

	var matches []string
	for i, loc := range indices {
		start := loc[0]
		end := len(body)
		if i+1 < len(indices) {
			end = indices[i+1][0]
		}
		block := body[start:end]
		for _, n := range needles {
			if strings.Contains(block, n) {
				matches = append(matches, block)
				break
			}
		}
	}

	return strings.Join(matches, "\n---\n")
}

func fetchText(url string) (string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "lark-cli/schema")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}
	var sb strings.Builder
	r := bufio.NewReader(resp.Body)
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return sb.String(), nil
}

// ListSchemaModules returns the currently supported llms-docs modules.
func ListSchemaModules() map[string]string {
	out := make(map[string]string, len(llmsModules))
	for k, v := range llmsModules {
		out[k] = v
	}
	return out
}
