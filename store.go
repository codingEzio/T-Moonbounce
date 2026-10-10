package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func validateSettings(s Settings) error {
	if s.Schema > 2 {
		return fmt.Errorf("this save needs a newer Moonbounce version")
	}
	if s.Audio.Track < 0 || s.Audio.Track >= len(tracks) || !finite(s.Audio.Volume) || s.Audio.Volume < 0 || s.Audio.Volume > 1 || s.Audio.PositionBytes < 0 {
		return fmt.Errorf("invalid audio settings")
	}
	j := s.Journey
	if j.Feel < 0 || j.Feel >= len(feels) || !finite(j.Steering) || !finite(j.DriftMeters) || j.DriftMeters < 0 || !finite(j.X) || !finite(j.Lateral) || !finite(j.Speed) || !finite(j.Offset) || !finite(j.Kilometers) || !finite(j.Boost) || j.Offset < 0 || j.Offset >= regionLength || j.Kilometers < 0 || j.Speed < 0 || j.Speed > 150 || j.Boost < 0 || j.Boost > 3 || j.X < -1.6 || j.X > 1.6 {
		return fmt.Errorf("invalid journey state")
	}
	return nil
}
func decodeSettings(b []byte, defaults Settings) (Settings, error) {
	s := defaults
	if err := json.Unmarshal(b, &s); err != nil {
		return defaults, err
	}
	if s.Schema < 2 {
		s.Schema = 2
		s.Typography = "Pixel"
		s.Cruise = true
	}
	return s, validateSettings(s)
}
func loadSettings(path string, defaults Settings) (Settings, string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaults, "", nil
	}
	if err != nil {
		return defaults, "", err
	}
	s, err := decodeSettings(b, defaults)
	if err == nil {
		var old struct{ Schema int }
		_ = json.Unmarshal(b, &old)
		if old.Schema < 2 {
			archive := path + ".pre-driving-v3"
			if _, e := os.Stat(archive); os.IsNotExist(e) {
				if e = writeAtomic(archive, b); e != nil {
					return defaults, "", fmt.Errorf("retain original save before migration: %w", e)
				}
			}
		}
		return s, "", nil
	}
	var newer struct{ Schema int }
	if json.Unmarshal(b, &newer) == nil && newer.Schema > 2 {
		return defaults, "", err
	}
	previous, e := os.ReadFile(path + ".previous")
	if e == nil {
		if recovered, e := decodeSettings(previous, defaults); e == nil {
			return recovered, "Recovered the previous local save; original retained.", nil
		}
	}
	return defaults, "Could not read the saved scene. Original retained; a fresh road is ready.", nil
}
func saveSettings(path string, s Settings) error {
	s.Schema = 2
	if err := validateSettings(s); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if old, e := os.ReadFile(path); e == nil {
		backup := path + ".previous"
		if _, e = decodeSettings(old, s); e != nil {
			backup = path + ".unreadable-" + time.Now().Format("20060102-150405.000000000")
		}
		if e = writeAtomic(backup, old); e != nil {
			return fmt.Errorf("retain previous save: %w", e)
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	return writeAtomic(path, b)
}
func writeAtomic(path string, b []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".moonbounce-write-")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(temp, path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	err = dir.Sync()
	closeErr = dir.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	got, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(got, b) {
		return fmt.Errorf("save readback differs")
	}
	return nil
}
