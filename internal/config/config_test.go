package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadConfig_MissingFile(t *testing.T) {
	_, err := ReadConfig(filepath.Join(t.TempDir(), "nonexistent.gob"))
	if err == nil {
		t.Fatal("ReadConfig: expected error for missing file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("ReadConfig: expected os.IsNotExist, got %v", err)
	}
}

func TestReadConfig_InvalidContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.gob")
	if err := os.WriteFile(path, []byte("not gob data"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("ReadConfig: expected error for invalid gob content")
	}
}

func TestWriteConfig_ThenRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.gob")

	c := DefaultConfig.Clone()
	c.Brightness = 200
	c.Gap = 5

	if err := WriteConfig(path, c); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	got, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("ReadConfig after write: %v", err)
	}
	if got.Brightness != 200 {
		t.Errorf("Brightness = %d, want 200", got.Brightness)
	}
	if got.Gap != 5 {
		t.Errorf("Gap = %d, want 5", got.Gap)
	}
}

func TestWriteConfig_CreateFails(t *testing.T) {
	// Write to a path under a non-existent directory so Create fails
	path := filepath.Join(t.TempDir(), "subdir", "config.gob")
	c := DefaultConfig.Clone()
	err := WriteConfig(path, c)
	if err == nil {
		t.Fatal("WriteConfig: expected error when parent dir missing")
	}
}

func TestClone_ReturnsCopy(t *testing.T) {
	orig := DefaultConfig.Clone()
	if orig == nil {
		t.Fatal("Clone: got nil")
	}

	clone := orig.Clone()
	if clone == nil {
		t.Fatal("Clone of clone: got nil")
	}
	if clone == orig {
		t.Fatal("Clone: returned same pointer as original")
	}

	clone.Brightness = 99
	clone.Tick.NumHours = 12
	if orig.Brightness == 99 || orig.Tick.NumHours == 12 {
		t.Error("Clone: modifying clone changed original")
	}
}

func TestClone_NilReceiver(t *testing.T) {
	var c *Config
	got := c.Clone()
	if got != nil {
		t.Errorf("Clone(nil) = %v, want nil", got)
	}
}

func TestClone_DefaultConfigNotMutated(t *testing.T) {
	// Using DefaultConfig.Clone() and mutating the result must not change DefaultConfig
	baseBrightness := DefaultConfig.Brightness
	baseGap := DefaultConfig.Gap

	c := DefaultConfig.Clone()
	c.Brightness = 255
	c.Gap = 10

	if DefaultConfig.Brightness != baseBrightness {
		t.Errorf("DefaultConfig.Brightness mutated: got %d, want %d", DefaultConfig.Brightness, baseBrightness)
	}
	if DefaultConfig.Gap != baseGap {
		t.Errorf("DefaultConfig.Gap mutated: got %d, want %d", DefaultConfig.Gap, baseGap)
	}
}

func TestReadConfig_ValidGob(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.gob")

	want := &Config{
		Version:     1,
		Brightness:  64,
		RefreshRate: 30 * time.Second,
		Gap:         2,
		Tick: TickConfig{
			TicksPerHour: 4,
			NumHours:     6,
			StartHour:    9,
		},
	}
	if err := WriteConfig(path, want); err != nil {
		t.Fatal(err)
	}

	got, err := ReadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Brightness != want.Brightness || got.RefreshRate != want.RefreshRate ||
		got.Gap != want.Gap || got.Tick.TicksPerHour != want.Tick.TicksPerHour ||
		got.Tick.NumHours != want.Tick.NumHours || got.Tick.StartHour != want.Tick.StartHour {
		t.Errorf("ReadConfig: got %+v, want %+v", got, want)
	}
}
