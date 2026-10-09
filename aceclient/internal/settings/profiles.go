package settings

import (
	"errors"
	"slices"
	"strings"
)

// Profile is a named connection profile. This is a Go-native addition: the C#
// installer has no saved connections (devices come from mDNS discovery or
// DlgManualIP each session). A profile deliberately has NO password field —
// the only password the C# remembers is the global LastDevicePassword slot
// governed by DlgDeviceLogin's "store password" box (see RecordDeviceLogin
// and LoginPrefill), and profiles do not add another place to keep one.
type Profile struct {
	// Name identifies the profile (unique, compared exactly after trimming).
	Name string `json:"Name"`
	// IP is the device address in IPAddress.ToString() form.
	IP string `json:"IP"`
	// Port is the HTTPS port (1..65535; the installer default is 443).
	Port int `json:"Port"`
	// User is the device user level (DeviceUserLevels ID, e.g. "admin"); may be empty.
	User string `json:"User"`
}

// ErrProfileName is returned for an empty profile name.
var ErrProfileName = errors.New("settings: profile name is empty")

// ErrProfilePort is returned for a port outside 1..65535.
var ErrProfilePort = errors.New("settings: profile port must be 1..65535")

// normalizeEndpoint validates and normalises p's address and port. The
// address goes through ParseIPAddress (same rules as DlgManualIP; the error
// is ErrInvalidIP).
func normalizeEndpoint(p Profile) (Profile, error) {
	addr, err := ParseIPAddress(p.IP)
	if err != nil {
		return Profile{}, err
	}
	p.IP = FormatIPAddress(addr)
	if p.Port < 1 || p.Port > ManualPortMax {
		return Profile{}, ErrProfilePort
	}
	return p, nil
}

// normalizeProfile additionally requires a non-empty (trimmed) name.
func normalizeProfile(p Profile) (Profile, error) {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return Profile{}, ErrProfileName
	}
	return normalizeEndpoint(p)
}

// Profiles returns the saved profiles in insertion order.
func (st *Store) Profiles() []Profile {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return slices.Clone(st.s.Profiles)
}

// Profile returns the profile with this name.
func (st *Store) Profile(name string) (Profile, bool) {
	name = strings.TrimSpace(name)
	st.mu.RLock()
	defer st.mu.RUnlock()
	for _, p := range st.s.Profiles {
		if p.Name == name {
			return p, true
		}
	}
	return Profile{}, false
}

// SaveProfile validates p and inserts it, or replaces (in place) the profile
// with the same name. Returns the stored, normalised profile.
func (st *Store) SaveProfile(p Profile) (Profile, error) {
	p, err := normalizeProfile(p)
	if err != nil {
		return Profile{}, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if i := slices.IndexFunc(st.s.Profiles, func(q Profile) bool { return q.Name == p.Name }); i >= 0 {
		st.s.Profiles[i] = p
	} else {
		st.s.Profiles = append(st.s.Profiles, p)
	}
	return p, st.saveLocked()
}

// DeleteProfile removes the profile with this name; removed is false when
// there is none.
func (st *Store) DeleteProfile(name string) (removed bool, err error) {
	name = strings.TrimSpace(name)
	st.mu.Lock()
	defer st.mu.Unlock()
	i := slices.IndexFunc(st.s.Profiles, func(q Profile) bool { return q.Name == name })
	if i < 0 {
		return false, nil
	}
	st.s.Profiles = slices.Delete(st.s.Profiles, i, i+1)
	return true, st.saveLocked()
}

// LastDevice returns the last-used device (Go-native; no password).
func (st *Store) LastDevice() (Profile, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if st.s.LastDevice == nil {
		return Profile{}, false
	}
	return *st.s.LastDevice, true
}

// SetLastDevice records the device the GUI connected to. Name may be empty
// (an ad-hoc connection); IP and Port are validated like SaveProfile.
func (st *Store) SetLastDevice(p Profile) (Profile, error) {
	p.Name = strings.TrimSpace(p.Name)
	p, err := normalizeEndpoint(p)
	if err != nil {
		return Profile{}, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s.LastDevice = &p
	return p, st.saveLocked()
}
