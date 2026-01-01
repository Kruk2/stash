package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
)

type JasnaPreset struct {
	Name                 string `json:"name"`
	MaxClipSize          int    `json:"maxClipSize"`
	TemporalOverlap      int    `json:"temporalOverlap"`
	SecondaryRestoration string `json:"secondaryRestoration"`
}

func getDefaultJasnaPresets() []JasnaPreset {
	return []JasnaPreset{
		{
			Name:                 "180s clip",
			MaxClipSize:          180,
			TemporalOverlap:      15,
			SecondaryRestoration: "",
		},
		{
			Name:                 "90s clip",
			MaxClipSize:          90,
			TemporalOverlap:      8,
			SecondaryRestoration: "",
		},
		{
			Name:                 "180s + UNet",
			MaxClipSize:          180,
			TemporalOverlap:      10,
			SecondaryRestoration: "unet-4x",
		},
		{
			Name:                 "180s + RTX",
			MaxClipSize:          180,
			TemporalOverlap:      10,
			SecondaryRestoration: "rtx-super-res",
		},
	}
}

func (i *Config) GetJasnaPresetsPath() string {
	configFileUsed := i.GetConfigFile()
	configDir := filepath.Dir(configFileUsed)
	fn := filepath.Join(configDir, "jasna-presets.json")
	return fn
}

func (i *Config) GetJasnaPresets() []JasnaPreset {
	fn := i.GetJasnaPresetsPath()

	exists, _ := fsutil.FileExists(fn)
	if !exists {
		return getDefaultJasnaPresets()
	}

	buf, err := os.ReadFile(fn)
	if err != nil {
		return getDefaultJasnaPresets()
	}

	var presets []JasnaPreset
	if err := json.Unmarshal(buf, &presets); err != nil {
		return getDefaultJasnaPresets()
	}

	if len(presets) == 0 {
		return getDefaultJasnaPresets()
	}

	return presets
}

func (i *Config) SetJasnaPresets(presets []JasnaPreset) {
	fn := i.GetJasnaPresetsPath()
	i.Lock()
	defer i.Unlock()

	buf, err := json.MarshalIndent(presets, "", "    ")
	if err != nil {
		logger.Warnf("error marshaling jasna presets: %v", err)
		return
	}

	if err := os.WriteFile(fn, buf, 0644); err != nil {
		logger.Warnf("error writing jasna presets: %v", err)
	}
}

func MakeSlug(name string) string {
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	slug = strings.Trim(slug, "-")
	return slug
}

func MakeUniqueSlug(slugs map[string]bool, name string) string {
	slug := MakeSlug(name)
	if !slugs[slug] {
		slugs[slug] = true
		return slug
	}

	i := 2
	for {
		uniqueSlug := slug + "-" + string(rune('0'+i))
		if !slugs[uniqueSlug] {
			slugs[uniqueSlug] = true
			return uniqueSlug
		}
		i++
	}
}
