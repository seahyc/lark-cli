package cmd

import (
	"errors"
	"github.com/yjwong/lark-cli/internal/api"
	"testing"
)

type fakeTaskSearcher struct {
	pages  []*api.TaskSearchPage
	tokens []string
	err    error
}

func (f *fakeTaskSearcher) SearchTasks(opts api.TaskSearchOptions) (*api.TaskSearchPage, error) {
	f.tokens = append(f.tokens, opts.PageToken)
	if f.err != nil {
		return nil, f.err
	}
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page, nil
}
func TestTaskSearchPagination(t *testing.T) {
	fake := &fakeTaskSearcher{pages: []*api.TaskSearchPage{{Items: []api.TaskSearchResult{{ID: "first"}}, HasMore: true, PageToken: "next"}, {Items: []api.TaskSearchResult{{ID: "second"}}, HasMore: false}}}
	result, err := collectTaskSearch(fake, api.TaskSearchOptions{}, 40)
	if err != nil || len(result.Items) != 2 || result.HasMore || len(fake.tokens) != 2 || fake.tokens[1] != "next" {
		t.Fatalf("result=%+v tokens=%v err=%v", result, fake.tokens, err)
	}
}
func TestTaskSearchPaginationBoundPreservesCursor(t *testing.T) {
	fake := &fakeTaskSearcher{pages: []*api.TaskSearchPage{{HasMore: true, PageToken: "next"}}}
	result, err := collectTaskSearch(fake, api.TaskSearchOptions{}, 1)
	if err != nil || !result.HasMore || result.PageToken != "next" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
func TestTaskSearchRejectsBrokenCursor(t *testing.T) {
	for _, token := range []string{"", "initial"} {
		fake := &fakeTaskSearcher{pages: []*api.TaskSearchPage{{HasMore: true, PageToken: token}}}
		if _, err := collectTaskSearch(fake, api.TaskSearchOptions{PageToken: "initial"}, 40); err == nil {
			t.Fatalf("accepted broken cursor %q", token)
		}
	}
}
func TestTaskSearchPropagatesError(t *testing.T) {
	want := errors.New("denied")
	fake := &fakeTaskSearcher{err: want}
	if _, err := collectTaskSearch(fake, api.TaskSearchOptions{}, 1); !errors.Is(err, want) {
		t.Fatalf("err=%v", err)
	}
}

func TestTaskOutputPreservesCanonicalIdentity(t *testing.T) {
	task := api.Task{TaskID: "t123", URL: "https://applink.larksuite.com/client/todo/detail?guid=synthetic&suite_entity_num=t123", Tasklists: []api.TasklistMembership{{TasklistGUID: "matching", SectionGUID: "others"}}}
	out := taskToOutput(task)
	if out.TaskID != task.TaskID || out.URL != task.URL || len(out.Tasklists) != 1 || out.Tasklists[0].TasklistGUID != "matching" {
		t.Fatalf("canonical identity lost: %+v", out)
	}
}
