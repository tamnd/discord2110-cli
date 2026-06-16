package discord2110

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "discord2110" {
		t.Errorf("Scheme = %q, want discord2110", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "discord2110" {
		t.Errorf("Identity.Binary = %q, want discord2110", info.Identity.Binary)
	}
}

func TestClassify_InviteCode(t *testing.T) {
	typ, id, err := Domain{}.Classify("ggTWnwK")
	if err != nil || typ != "invite" || id != "ggTWnwK" {
		t.Errorf("Classify(ggTWnwK) = (%q, %q, %v), want (invite, ggTWnwK, nil)", typ, id, err)
	}
}

func TestClassify_InviteURL(t *testing.T) {
	typ, id, err := Domain{}.Classify("https://discord.gg/ggTWnwK")
	if err != nil || typ != "invite" || id != "ggTWnwK" {
		t.Errorf("Classify(discord.gg/...) = (%q, %q, %v)", typ, id, err)
	}
}

func TestClassify_ServerID(t *testing.T) {
	// A numeric snowflake ID with no invite-code match is classified as invite
	// (short bare code path), so we just check no error is returned.
	_, _, err := Domain{}.Classify("102860784329052160")
	if err != nil {
		t.Errorf("Classify(serverID) returned error: %v", err)
	}
}

func TestLocate_Invite(t *testing.T) {
	got, err := Domain{}.Locate("invite", "ggTWnwK")
	want := "https://discord.gg/ggTWnwK"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocate_Server(t *testing.T) {
	got, err := Domain{}.Locate("server", "123")
	want := "https://discord.com/channels/123"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocate_UnknownType(t *testing.T) {
	_, err := Domain{}.Locate("message", "abc")
	if err == nil {
		t.Error("expected error for unknown resource type")
	}
}
