package presets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DlgPresets UI strings (ACEServiceInstaller/ICUServiceInstaller/DlgPresets.cs).
const (
	PresetsDialogTitle  = "Property Presets " + AppName
	PresetsDialogPrompt = "Please select a property preset from the list below."
)

// PresetColumns are the DlgPresets list column headers, in order.
var PresetColumns = []string{"Name", "Version", "Model", "Sockets", "Power", "Information"}

// PowerPropertyID is the literal Id DlgPresets reads the "Power" column from
// (0x2129 sub 0, mainNormalMaxCurrent). Note that SaveProperties writes ids
// as "2129_00", so files saved by the installer show no power.
const PowerPropertyID = "2129_0"

// PresetEntry is one DlgPresets list row (the ListStore data fields).
type PresetEntry struct {
	Name        string    // Path.GetFileNameWithoutExtension
	Version     string    // Settings/Version
	Model       string    // Settings/Model
	Sockets     string    // Settings/NumberOfSockets
	Power       string    // Value of the last Settings/Properties/Property with Id == "2129_0"
	Information string    // Settings/Information
	Path        string    // FileInfo.FullName: pass to LoadProperties on OK
	ModTime     time.Time // FileInfo.LastWriteTime (sort key)
}

// Columns returns the row values in PresetColumns order.
func (p PresetEntry) Columns() []string {
	return []string{p.Name, p.Version, p.Model, p.Sockets, p.Power, p.Information}
}

// getElementString ports DlgPresets.GetElementString.
func getElementString(e *xElement, name string) string {
	if c := e.Element(name); c != nil {
		return c.Value()
	}
	return ""
}

// LoadPresetsFromFolder ports DlgPresets.LoadPresetsFromFolder: the folder is
// created when missing; its *.xml and *.iip files (extension compared
// case-insensitively; .exml is not listed) are read newest first
// (LastWriteTime descending, stable) as plain "Settings" XML.
//
// As in the C#, the first file that fails (unreadable, malformed XML, a
// <Settings> without <Properties>, a <Property> without Id, or the power
// property without Value) stops the scan of this folder: the rows read so far
// are returned with the error (the C# logs it at Debug level). The C# also
// leaves an empty row for the failing file; that row is not reproduced.
func LoadPresetsFromFolder(path string) ([]PresetEntry, error) {
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, err
		}
	}
	dirents, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	type file struct {
		name    string
		full    string
		modTime time.Time
	}
	var files []file
	for _, de := range dirents {
		full := filepath.Join(path, de.Name())
		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			continue // Directory.GetFiles lists files only
		}
		ext := strings.ToLower(netGetExtension(de.Name()))
		if ext != ".xml" && ext != ".iip" {
			continue
		}
		if abs, err := filepath.Abs(full); err == nil {
			full = abs
		}
		files = append(files, file{name: de.Name(), full: full, modTime: info.ModTime()})
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].modTime.After(files[j].modTime) })

	var out []PresetEntry
	for _, f := range files {
		entry := PresetEntry{
			Name:    strings.TrimSuffix(f.name, filepath.Ext(f.name)),
			Path:    f.full,
			ModTime: f.modTime,
		}
		raw, err := os.ReadFile(f.full)
		if err != nil {
			return out, err
		}
		doc, err := xDocumentLoad(raw)
		if err != nil {
			return out, fmt.Errorf("%s: %w", f.full, err)
		}
		if settings := doc.Element("Settings"); settings != nil {
			entry.Version = getElementString(settings, "Version")
			entry.Model = getElementString(settings, "Model")
			entry.Sockets = getElementString(settings, "NumberOfSockets")
			entry.Information = getElementString(settings, "Information")
			props := settings.Element("Properties")
			if props == nil {
				return out, fmt.Errorf("%s: %w", f.full, errMissing("<Properties>"))
			}
			for _, item := range props.Elements("Property") {
				id, ok := item.Attribute("Id")
				if !ok {
					return out, fmt.Errorf("%s: %w", f.full, errMissing("Property/@Id"))
				}
				if id == PowerPropertyID {
					v, ok := item.Attribute("Value")
					if !ok {
						return out, fmt.Errorf("%s: %w", f.full, errMissing("Property/@Value"))
					}
					entry.Power = v
				}
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

// FillPresetList ports DlgPresets.FillPresetList: LoadPresetsFromFolder over
// f.PresetListFolders() in order, concatenated. A folder error does not stop
// the other folders; all are returned joined (non-fatal, logged in the C#).
// The dialog enables the list only when it has rows and selects row 0.
func FillPresetList(f Folders) ([]PresetEntry, error) {
	var out []PresetEntry
	var errs []error
	for _, dir := range f.PresetListFolders() {
		rows, err := LoadPresetsFromFolder(dir)
		out = append(out, rows...)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return out, errors.Join(errs...)
}
