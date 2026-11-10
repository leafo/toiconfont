// Package selection reads IcoMoon compatible selection.json files.
package selection

import (
	"encoding/json"
	"fmt"
	"os"
)

type Icon struct {
	Name  string
	Code  rune
	Width float64
	Paths []string
}

type File struct {
	Family   string
	Prefix   string
	Height   float64
	EmSize   float64
	Baseline float64 // percent of the em below the baseline
	Icons    []Icon
}

type rawFile struct {
	Height float64 `json:"height"`
	Icons  []struct {
		Icon struct {
			Paths []string `json:"paths"`
			Width float64  `json:"width"`
		} `json:"icon"`
		Properties struct {
			Name string `json:"name"`
			Code rune   `json:"code"`
		} `json:"properties"`
	} `json:"icons"`
	Preferences struct {
		FontPref struct {
			Prefix   string `json:"prefix"`
			Metadata struct {
				FontFamily string `json:"fontFamily"`
			} `json:"metadata"`
			Metrics struct {
				EmSize   float64  `json:"emSize"`
				Baseline *float64 `json:"baseline"`
			} `json:"metrics"`
		} `json:"fontPref"`
	} `json:"preferences"`
}

func Read(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw rawFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	pref := raw.Preferences.FontPref
	f := &File{
		Family:   pref.Metadata.FontFamily,
		Prefix:   pref.Prefix,
		Height:   raw.Height,
		EmSize:   pref.Metrics.EmSize,
		Baseline: 6.25,
	}
	if f.Family == "" {
		f.Family = "icons"
	}
	if f.Prefix == "" {
		f.Prefix = "icon-"
	}
	if f.Height == 0 {
		f.Height = 1024
	}
	if f.EmSize == 0 {
		f.EmSize = f.Height
	}
	if pref.Metrics.Baseline != nil {
		f.Baseline = *pref.Metrics.Baseline
	}

	names := map[string]bool{}
	codes := map[rune]string{}
	for i, r := range raw.Icons {
		icon := Icon{Name: r.Properties.Name, Code: r.Properties.Code, Width: r.Icon.Width, Paths: r.Icon.Paths}
		if icon.Width == 0 {
			icon.Width = f.Height
		}
		switch {
		case icon.Name == "":
			return nil, fmt.Errorf("%s: icon %d has no name", path, i)
		case icon.Code <= 0:
			return nil, fmt.Errorf("%s: icon %s has no code", path, icon.Name)
		case names[icon.Name]:
			return nil, fmt.Errorf("%s: duplicate icon name %s", path, icon.Name)
		case codes[icon.Code] != "":
			return nil, fmt.Errorf("%s: %s and %s share code %x", path, codes[icon.Code], icon.Name, icon.Code)
		}
		names[icon.Name] = true
		codes[icon.Code] = icon.Name
		f.Icons = append(f.Icons, icon)
	}
	if len(f.Icons) == 0 {
		return nil, fmt.Errorf("%s: no icons", path)
	}
	return f, nil
}
