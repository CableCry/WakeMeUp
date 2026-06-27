package main

import (
	"bytes"
	"net"
	"reflect"
	"testing"
)

func TestNormalizeMAC(t *testing.T) {
	good := map[string]string{
		"AA:BB:CC:DD:EE:FF":     "aa:bb:cc:dd:ee:ff",
		"aa-bb-cc-dd-ee-ff":     "aa:bb:cc:dd:ee:ff",
		"  00:11:22:33:44:55  ": "00:11:22:33:44:55",
	}
	for in, want := range good {
		got, err := normalizeMAC(in)
		if err != nil {
			t.Errorf("normalizeMAC(%q) unexpected error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("normalizeMAC(%q) = %q, want %q", in, got, want)
		}
	}

	bad := []string{"", "not-a-mac", "AA:BB:CC", "00:11:22:33:44:55:66:77"}
	for _, in := range bad {
		if _, err := normalizeMAC(in); err == nil {
			t.Errorf("normalizeMAC(%q) expected error, got nil", in)
		}
	}
}

func TestDeviceTarget(t *testing.T) {
	cases := []struct {
		dev  Device
		want string
	}{
		{Device{MAC: "00:11:22:33:44:55"}, "255.255.255.255:9"},
		{Device{IP: "192.168.1.10", Port: 7}, "192.168.1.10:7"},
		{Device{IP: "10.0.0.255"}, "10.0.0.255:9"},
		{Device{Port: 4000}, "255.255.255.255:4000"},
	}
	for _, c := range cases {
		if got := c.dev.target(); got != c.want {
			t.Errorf("target() = %q, want %q", got, c.want)
		}
	}
}

func TestBuildMagicPacket(t *testing.T) {
	hw, err := net.ParseMAC("00:11:22:33:44:55")
	if err != nil {
		t.Fatal(err)
	}
	p := buildMagicPacket(hw)
	if len(p) != 102 {
		t.Fatalf("packet length = %d, want 102", len(p))
	}
	for i := 0; i < 6; i++ {
		if p[i] != 0xFF {
			t.Fatalf("sync byte %d = %#x, want 0xFF", i, p[i])
		}
	}
	for i := 0; i < 16; i++ {
		off := 6 + i*6
		if !bytes.Equal(p[off:off+6], hw) {
			t.Fatalf("repetition %d does not match MAC", i)
		}
	}
}

func TestStorageRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AppData", tmp)         // Windows
	t.Setenv("XDG_CONFIG_HOME", tmp) // Unix

	// Missing file yields an empty list, not an error.
	got, err := LoadDevices()
	if err != nil {
		t.Fatalf("LoadDevices on empty: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no devices, got %d", len(got))
	}

	devs := []Device{
		{Name: "A", MAC: "00:11:22:33:44:55"},
		{Name: "B", MAC: "aa:bb:cc:dd:ee:ff", IP: "1.2.3.4", Port: 7},
	}
	if err := SaveDevices(devs); err != nil {
		t.Fatalf("SaveDevices: %v", err)
	}
	got, err = LoadDevices()
	if err != nil {
		t.Fatalf("LoadDevices: %v", err)
	}
	if !reflect.DeepEqual(got, devs) {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", got, devs)
	}
}

func TestFindDevice(t *testing.T) {
	devs := []Device{{Name: "Desktop"}, {Name: "Media Server"}}
	if i := findDevice(devs, "desktop"); i != 0 {
		t.Errorf("case-insensitive match = %d, want 0", i)
	}
	if i := findDevice(devs, "  Media Server "); i != 1 {
		t.Errorf("trimmed match = %d, want 1", i)
	}
	if i := findDevice(devs, "nope"); i != -1 {
		t.Errorf("no match = %d, want -1", i)
	}
}

// drive the form's submit logic directly to validate without simulating typed
// runes through the terminal layer.
func TestSubmitFormValidation(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := newModel(nil)
	m.state = stateForm
	m.editIndex = -1

	// Missing name.
	m.inputs[fieldName].SetValue("")
	m.inputs[fieldMAC].SetValue("00:11:22:33:44:55")
	res, _ := m.submitForm()
	mm := res.(model)
	if mm.formErr == "" {
		t.Error("expected error for missing name")
	}
	if len(mm.devices) != 0 {
		t.Error("device should not be saved when invalid")
	}

	// Invalid MAC.
	m.inputs[fieldName].SetValue("PC")
	m.inputs[fieldMAC].SetValue("garbage")
	res, _ = m.submitForm()
	mm = res.(model)
	if mm.formErr == "" {
		t.Error("expected error for invalid MAC")
	}

	// Invalid port.
	m.inputs[fieldMAC].SetValue("00:11:22:33:44:55")
	m.inputs[fieldPort].SetValue("99999")
	res, _ = m.submitForm()
	mm = res.(model)
	if mm.formErr == "" {
		t.Error("expected error for out-of-range port")
	}

	// Valid: should persist and normalize the MAC.
	m.inputs[fieldPort].SetValue("")
	m.inputs[fieldMAC].SetValue("AA-BB-CC-DD-EE-FF")
	res, _ = m.submitForm()
	mm = res.(model)
	if mm.formErr != "" {
		t.Fatalf("unexpected form error: %s", mm.formErr)
	}
	if len(mm.devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(mm.devices))
	}
	if mm.devices[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MAC not normalized: %q", mm.devices[0].MAC)
	}
	if mm.state != stateList {
		t.Error("should return to list after a successful save")
	}
}

func TestSubmitFormRejectsDuplicateName(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := newModel([]Device{{Name: "PC", MAC: "00:11:22:33:44:55"}})
	m.state = stateForm
	m.editIndex = -1 // adding a new one
	m.inputs[fieldName].SetValue("pc")
	m.inputs[fieldMAC].SetValue("aa:bb:cc:dd:ee:ff")
	res, _ := m.submitForm()
	mm := res.(model)
	if mm.formErr == "" {
		t.Error("expected duplicate-name error")
	}
	if len(mm.devices) != 1 {
		t.Errorf("duplicate should not be added, have %d devices", len(mm.devices))
	}
}

func TestEditAllowsSameName(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := newModel([]Device{{Name: "PC", MAC: "00:11:22:33:44:55"}})
	m.state = stateForm
	m.editIndex = 0 // editing the existing one
	m.inputs[fieldName].SetValue("PC")
	m.inputs[fieldMAC].SetValue("aa:bb:cc:dd:ee:ff")
	res, _ := m.submitForm()
	mm := res.(model)
	if mm.formErr != "" {
		t.Fatalf("editing should allow keeping the same name: %s", mm.formErr)
	}
	if mm.devices[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("edit did not apply: %q", mm.devices[0].MAC)
	}
}

func TestDeleteSelected(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := newModel([]Device{
		{Name: "A", MAC: "00:11:22:33:44:55"},
		{Name: "B", MAC: "aa:bb:cc:dd:ee:ff"},
	})
	m.deleteIndex = 0
	res, _ := m.deleteSelected()
	mm := res.(model)
	if len(mm.devices) != 1 {
		t.Fatalf("expected 1 device after delete, got %d", len(mm.devices))
	}
	if mm.devices[0].Name != "B" {
		t.Errorf("wrong device remained: %s", mm.devices[0].Name)
	}
	if mm.state != stateList {
		t.Error("should return to list after delete")
	}
}

func TestViewRendersAllStates(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// Empty list, then a populated one across every state. View must never panic.
	for _, devs := range [][]Device{nil, {{Name: "PC", MAC: "00:11:22:33:44:55"}}} {
		m := newModel(devs)
		m.width, m.height = 80, 24
		m.setSize()
		for _, st := range []uiState{stateList, stateForm, stateConfirm} {
			m.state = st
			m.deleteIndex = 0
			if v := m.View(); v.Content == "" {
				t.Errorf("empty render for state %d (devices=%d)", st, len(devs))
			}
		}
	}
}

func TestWakeAnimation(t *testing.T) {
	m := newModel([]Device{{Name: "PC", MAC: "00:11:22:33:44:55"}})
	m.animGen = 1
	*m.anim = wakeAnim{active: true, name: "PC"}

	// A successful reply is recorded on the shared animation state.
	res, _ := m.Update(wolResultMsg{name: "PC", err: nil})
	m = res.(model)
	if !m.anim.gotReply || m.anim.failed {
		t.Fatalf("expected successful reply recorded, got %+v", *m.anim)
	}

	// Ticks advance the fill until it is full.
	guard := 0
	for m.anim.percent < 1 {
		res, _ = m.Update(wakeTickMsg{gen: 1})
		m = res.(model)
		if guard++; guard > 100 {
			t.Fatal("fill never completed")
		}
	}

	// While full, the animation stays active until the clear fires.
	res, _ = m.Update(wakeTickMsg{gen: 1})
	m = res.(model)
	if !m.anim.active {
		t.Error("anim should stay active until clear fires")
	}

	res, _ = m.Update(clearAnimMsg{gen: 1})
	m = res.(model)
	if m.anim.active {
		t.Error("clearAnimMsg should deactivate the animation")
	}

	// A stale-generation tick must be ignored.
	*m.anim = wakeAnim{active: true, name: "PC", percent: 0.5}
	res, _ = m.Update(wakeTickMsg{gen: 0})
	m = res.(model)
	if m.anim.percent != 0.5 {
		t.Error("stale tick should be ignored")
	}
}
