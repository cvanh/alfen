package i18n

import (
	"embed"
	"io/fs"
	"os"
	"strings"
	"sync"
)

// DefaultFolder ports Tooltips.m_tooltipFolder: the installer reads
// "tooltip_*_*.csv" from a "Tooltips" folder relative to its working
// directory. NewFolder(DefaultFolder) reproduces that; the package default
// (Default) reads the embedded copy instead.
const DefaultFolder = "Tooltips"

// embedded holds a byte-identical copy of the installer's only shipped
// tooltip file (firmware/msi_work/files3/tooltip_en_GB.csv).
//
//go:embed tooltip_en_GB.csv
var embedded embed.FS

// Tooltips ports the static class ICUServiceInstaller.Tooltips (Tooltips.cs)
// as an instance so that the source of the CSV files can be chosen. It is
// safe for concurrent use. Collections returned by it are shared and must be
// treated as read-only.
//
// Create it with NewFS, NewFolder or NewEmbedded. The zero value (and
// NewFS(nil)) has no source: it behaves like a missing "Tooltips" folder,
// so every lookup returns "". The zero value's current language is
// LanguageUnknown rather than English until SetLanguage is called.
type Tooltips struct {
	fsys fs.FS

	mu          sync.Mutex
	loaded      bool // m_lsTooltip != null
	collections []*TooltipCollection
	current     Language // m_currentLanguage
}

// NewFS returns a Tooltips that loads every "tooltip_*_*.csv" file in the
// root of fsys on first use. The current language starts as English, as
// Tooltips.m_currentLanguage does.
func NewFS(fsys fs.FS) *Tooltips {
	return &Tooltips{fsys: fsys, current: LanguageEnglish}
}

// NewFolder returns a Tooltips reading from a directory on disk, like the
// installer's "Tooltips" folder (see DefaultFolder). A missing directory
// yields no collections, as the C# Directory.Exists check does.
func NewFolder(dir string) *Tooltips {
	return NewFS(os.DirFS(dir))
}

// NewEmbedded returns a Tooltips backed by the embedded tooltip_en_GB.csv.
func NewEmbedded() *Tooltips {
	return NewFS(embedded)
}

// Default is the process-wide instance used by the package-level helpers,
// standing in for the C# static class. It reads the embedded tooltip_en_GB.csv.
// To use another source (e.g. NewFolder(DefaultFolder), as the installer
// does), either reassign Default once at start-up, before any lookup can run
// concurrently, or call the methods of your own *Tooltips instead of the
// package-level helpers. Reassigning it later is a data race.
var Default = NewEmbedded()

// Language ports the Tooltips.Language getter.
func (t *Tooltips) Language() Language {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.current
}

// SetLanguage ports the Tooltips.Language setter. Like the C#, any value is
// accepted; setting LanguageUnknown makes every default-language lookup miss.
func (t *Tooltips) SetLanguage(l Language) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current = l
}

// Init ports Tooltips.InitToolTip: load the files once.
func (t *Tooltips) Init() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.initLocked()
}

func (t *Tooltips) initLocked() {
	if !t.loaded {
		t.reinitLocked()
	}
}

// Reinit ports Tooltips.ReinitToolTip: drop everything and reload all
// "tooltip_*_*.csv" files. Files whose name has no known language, or that
// fail to read, are skipped (ParseFile returns null for them).
func (t *Tooltips) Reinit() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reinitLocked()
}

func (t *Tooltips) reinitLocked() {
	t.collections = nil
	t.loaded = true
	if t.fsys == nil {
		return // zero value / NewFS(nil): same as !Directory.Exists
	}
	entries, err := fs.ReadDir(t.fsys, ".") // sorted by name
	if err != nil {
		return // !Directory.Exists(m_tooltipFolder)
	}
	for _, e := range entries {
		if e.IsDir() || !matchTooltipFile(e.Name()) {
			continue
		}
		if c, err := ParseTooltipFile(t.fsys, e.Name()); err == nil {
			t.collections = append(t.collections, c)
		}
	}
}

// ParseTooltipFile ports Tooltips.ParseFile (Tooltips.cs): the language comes
// from the file name (LanguageFromFileName); ErrUnknownLanguage is returned
// before the file is opened if it maps to none.
func ParseTooltipFile(fsys fs.FS, name string) (*TooltipCollection, error) {
	lang := LanguageFromFileName(name)
	if lang == LanguageUnknown {
		return nil, ErrUnknownLanguage
	}
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseTooltips(f, lang)
}

// Collection ports Tooltips.GetCollection: lazy-loads, maps LanguageUnknown
// to the current language and returns the FIRST collection of that language,
// or nil.
func (t *Tooltips) Collection(lang Language) *TooltipCollection {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.initLocked()
	if lang == LanguageUnknown {
		lang = t.current
	}
	for _, c := range t.collections {
		if c.Language == lang {
			return c
		}
	}
	return nil
}

// GetTooltip ports Tooltips.GetTooltip: the tooltip text for tooltipID with
// every literal `\n` replaced by a newline, or "" when the language has no
// collection or the ID is absent. Pass LanguageUnknown for the current
// language (the C# default argument).
func (t *Tooltips) GetTooltip(tooltipID string, lang Language) string {
	c := t.Collection(lang)
	i := c.index(tooltipID)
	if i < 0 {
		return ""
	}
	return strings.ReplaceAll(c.Tips[i].Tooltip, `\n`, "\n")
}

// GetLabelText ports Tooltips.GetLabelText: the label column for tooltipID,
// returned verbatim (no `\n` replacement), or "". Pass LanguageUnknown for
// the current language.
func (t *Tooltips) GetLabelText(tooltipID string, lang Language) string {
	c := t.Collection(lang)
	i := c.index(tooltipID)
	if i < 0 {
		return ""
	}
	return c.Tips[i].LabelText
}

// HasTooltip ports the information-button rule in the UIPropertyBase
// constructors (UIPropertyBase.cs), on this instance:
//
//	m_tooltipText = Tooltips.GetTooltip(TooltipID);
//	m_btnTooltip.Visible = !string.IsNullOrEmpty(m_tooltipText);
//
// The C# then shows the text in DlgInfo when the button is clicked
// (OnBtnTooltipClicked).
func (t *Tooltips) HasTooltip(tooltipID string) bool {
	return t.GetTooltip(tooltipID, LanguageUnknown) != ""
}

// MakeTooltip ports UIPropertyBase.MakeToolTip(text) (UIPropertyBase.cs) on
// this instance: an empty text stays empty; otherwise the text is ignored
// and the tooltip of tooltipID in the current language is returned
// (possibly "").
func (t *Tooltips) MakeTooltip(text, tooltipID string) string {
	t.Init() // Tooltips.InitToolTip()
	if text == "" {
		return text
	}
	return t.GetTooltip(tooltipID, LanguageUnknown)
}

// Tooltip is Tooltips.GetTooltip(tooltipID) on Default in the current
// language — what every UIProperty* control calls with its TooltipID.
func Tooltip(tooltipID string) string {
	return Default.GetTooltip(tooltipID, LanguageUnknown)
}

// LabelText is Tooltips.GetLabelText(tooltipID) on Default in the current
// language.
func LabelText(tooltipID string) string {
	return Default.GetLabelText(tooltipID, LanguageUnknown)
}

// SetLanguage sets Default's current language (Tooltips.Language = l).
func SetLanguage(l Language) { Default.SetLanguage(l) }

// CurrentLanguage returns Default's current language (Tooltips.Language).
func CurrentLanguage() Language { return Default.Language() }
