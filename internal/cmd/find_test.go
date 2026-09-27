package cmd

import (
	"reflect"
	"testing"
)

func TestNormalizeFindTypes(t *testing.T) {
	got, err := normalizeFindTypes([]string{"messages,docs", "messages", "PEOPLE"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"messages", "docs", "people"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeFindTypes() = %#v, want %#v", got, want)
	}
}

func TestNormalizeFindTypesDefaults(t *testing.T) {
	got, err := normalizeFindTypes(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"messages", "docs", "people", "chats"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeFindTypes() = %#v, want %#v", got, want)
	}
}

func TestNormalizeFindTypesRejectsUnknown(t *testing.T) {
	if _, err := normalizeFindTypes([]string{"meetings"}); err == nil {
		t.Fatal("normalizeFindTypes() error = nil, want error")
	}
}

func TestCapFindResults(t *testing.T) {
	got := capFindResults([]int{1, 2, 3}, 2)
	if !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("capFindResults() = %#v", got)
	}
}
