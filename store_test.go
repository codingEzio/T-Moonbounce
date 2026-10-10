package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveReopenAndRecovery(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	s := Settings{Schema: 2, Language: "繁體中文", Typography: "Pixel", Journey: newJourney(88)}
	s.Journey.Region = 50
	s.Journey.Offset = 123.75
	s.Journey.Kilometers = 73.12
	if err := saveSettings(p, s); err != nil {
		t.Fatal(err)
	}
	got, warning, err := loadSettings(p, Settings{})
	if err != nil || warning != "" || got.Journey != s.Journey || got.Language != s.Language {
		t.Fatalf("reopen: %+v %s %v", got, warning, err)
	}
	s.Journey.Offset = 140
	if err := saveSettings(p, s); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	got, warning, err = loadSettings(p, Settings{})
	if err != nil || warning == "" || got.Journey.Offset != 123.75 {
		t.Fatalf("recovery failed: %+v %s %v", got, warning, err)
	}
	if err := saveSettings(p, got); err != nil {
		t.Fatal(err)
	}
	retained, _ := filepath.Glob(p + ".unreadable-*")
	if len(retained) != 1 {
		t.Fatal("unreadable original not preserved")
	}
}
func TestFutureSaveIsNotOverwritten(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	_ = os.WriteFile(p, []byte(`{"Schema":99}`), 0600)
	_, _, err := loadSettings(p, Settings{})
	if err == nil {
		t.Fatal("future schema accepted")
	}
}
