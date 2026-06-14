package cmo

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "cmo" {
		t.Errorf("Scheme = %q, want cmo", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "cmo" {
		t.Errorf("Identity.Binary = %q, want cmo", info.Identity.Binary)
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("editions", "all")
	want := BaseURL + archivePath
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("nosuchtype", "x")
	if err == nil {
		t.Error("expected error for unknown type")
	}
}
