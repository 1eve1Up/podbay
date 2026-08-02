package receipt

import (
	"testing"
	"time"
)

func receiptFixture(project, contract string, profiles []string, svcs []ServiceRecord) *Receipt {
	return &Receipt{
		FormatVersion: CurrentFormatVersion,
		GeneratedAt:   time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		ContractPath:  contract,
		Project:       project,
		Profiles:      profiles,
		Services:      svcs,
	}
}

func TestCompareReceipts_identical(t *testing.T) {
	a := receiptFixture("p", "/app/p.yaml", []string{"a", "b"}, []ServiceRecord{
		{Service: "api", ContainerName: "c1", Image: "img:v1"},
	})
	b := receiptFixture("p", "/app/p.yaml", []string{"b", "a"}, []ServiceRecord{
		{Service: "api", ContainerName: "c1", Image: "img:v1"},
	})
	got := CompareReceipts(a, b)
	if got.Drift {
		t.Fatalf("expected no drift, got %+v", got)
	}
	if len(got.Services) != 0 {
		t.Fatalf("services: %+v", got.Services)
	}
	if !got.ProjectMatch || !got.ContractPathMatch || !got.ProfilesMatch {
		t.Fatalf("matches: %+v", got)
	}
}

func TestCompareReceipts_projectMismatch(t *testing.T) {
	a := receiptFixture("p1", "/x.yaml", nil, nil)
	b := receiptFixture("p2", "/x.yaml", nil, nil)
	got := CompareReceipts(a, b)
	if !got.Drift || got.ProjectMatch {
		t.Fatalf("got %+v", got)
	}
	if got.ProjectA != "p1" || got.ProjectB != "p2" {
		t.Fatal(got)
	}
}

func TestCompareReceipts_contractPathMismatch(t *testing.T) {
	a := receiptFixture("p", "/a.yaml", nil, nil)
	b := receiptFixture("p", "/b.yaml", nil, nil)
	got := CompareReceipts(a, b)
	if !got.Drift || got.ContractPathMatch {
		t.Fatalf("got %+v", got)
	}
}

func TestCompareReceipts_contractDigestMismatch(t *testing.T) {
	a := receiptFixture("p", "/x.yaml", nil, nil)
	b := receiptFixture("p", "/x.yaml", nil, nil)
	a.ContractDigest = "sha256:aaa"
	b.ContractDigest = "sha256:bbb"
	got := CompareReceipts(a, b)
	if !got.Drift || got.ContractDigestMatch || !got.ContractDigestComparable {
		t.Fatalf("got %+v", got)
	}
}

func TestCompareReceipts_contractDigestIncomparable(t *testing.T) {
	a := receiptFixture("p", "/x.yaml", nil, nil)
	b := receiptFixture("p", "/x.yaml", nil, nil)
	a.ContractDigest = "sha256:aaa"
	got := CompareReceipts(a, b)
	if got.Drift {
		t.Fatalf("incomparable must not set drift: %+v", got)
	}
	if got.ContractDigestMatch || got.ContractDigestComparable {
		t.Fatalf("got %+v", got)
	}
}

func TestCompareReceipts_contractDigestBothEmptyMatch(t *testing.T) {
	a := receiptFixture("p", "/x.yaml", nil, nil)
	b := receiptFixture("p", "/x.yaml", nil, nil)
	got := CompareReceipts(a, b)
	if got.Drift || !got.ContractDigestMatch || !got.ContractDigestComparable {
		t.Fatalf("got %+v", got)
	}
}

func TestCompareReceipts_profilesMismatch(t *testing.T) {
	a := receiptFixture("p", "/x.yaml", []string{"dev"}, nil)
	b := receiptFixture("p", "/x.yaml", []string{"prod"}, nil)
	got := CompareReceipts(a, b)
	if !got.Drift || got.ProfilesMatch {
		t.Fatalf("got %+v", got)
	}
}

func TestCompareReceipts_serviceAddedRemoved(t *testing.T) {
	a := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "api", ContainerName: "n1"},
	})
	b := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "web", ContainerName: "n2"},
	})
	got := CompareReceipts(a, b)
	if !got.Drift || len(got.Services) != 2 {
		t.Fatalf("got %+v", got)
	}
	var seenAdd, seenRem bool
	for _, s := range got.Services {
		for _, c := range s.Codes {
			if c == CodeServiceRemoved {
				seenRem = seenRem || s.Service == "api"
			}
			if c == CodeServiceAdded {
				seenAdd = seenAdd || s.Service == "web"
			}
		}
	}
	if !seenAdd || !seenRem {
		t.Fatalf("got %+v", got.Services)
	}
}

func TestCompareReceipts_fieldChanges(t *testing.T) {
	a := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "api", ContainerName: "n1", Image: "i1", ContainerID: "id1"},
	})
	b := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "api", ContainerName: "n2", Image: "i2", ContainerID: "id2"},
	})
	got := CompareReceipts(a, b)
	if !got.Drift || len(got.Services) != 1 {
		t.Fatalf("got %+v", got)
	}
	sd := got.Services[0]
	if sd.Service != "api" {
		t.Fatal(sd)
	}
	want := []string{CodeImageChanged, CodeContainerNameChanged, CodeContainerIDChanged}
	if len(sd.Codes) != len(want) {
		t.Fatalf("codes %v want %v", sd.Codes, want)
	}
	for i, w := range want {
		if sd.Codes[i] != w {
			t.Fatalf("codes %v", sd.Codes)
		}
	}
}

func TestCompareReceipts_envChanged(t *testing.T) {
	e1 := []EnvVar{{Name: "A", Value: "1"}}
	e2 := []EnvVar{{Name: "A", Value: "2"}}
	a := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "api", ContainerName: "n1", Env: &e1},
	})
	b := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "api", ContainerName: "n1", Env: &e2},
	})
	got := CompareReceipts(a, b)
	if !got.Drift || len(got.Services) != 1 {
		t.Fatalf("got %+v", got)
	}
	if got.Services[0].Codes[0] != CodeEnvChanged {
		t.Fatalf("codes %v", got.Services[0].Codes)
	}
}

func TestCompareReceipts_envIncomparableNoDrift(t *testing.T) {
	e := []EnvVar{{Name: "A", Value: "1"}}
	a := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "api", ContainerName: "n1", Env: &e},
	})
	b := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "api", ContainerName: "n1"},
	})
	got := CompareReceipts(a, b)
	if got.Drift {
		t.Fatalf("expected no drift, got %+v", got)
	}
	if len(got.Services) != 1 || got.Services[0].Codes[0] != CodeEnvIncomparable {
		t.Fatalf("got %+v", got.Services)
	}
}

func TestCompareReceipts_mountsChanged(t *testing.T) {
	m1 := []MountSpec{{Type: "bind", Source: "/a", Destination: "/b"}}
	m2 := []MountSpec{{Type: "bind", Source: "/a", Destination: "/c"}}
	a := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "api", ContainerName: "n1", Mounts: &m1},
	})
	b := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "api", ContainerName: "n1", Mounts: &m2},
	})
	got := CompareReceipts(a, b)
	if !got.Drift || len(got.Services) != 1 {
		t.Fatalf("got %+v", got)
	}
	if got.Services[0].Codes[0] != CodeMountsChanged {
		t.Fatalf("codes %v", got.Services[0].Codes)
	}
}

func TestCompareReceipts_nilPanics(t *testing.T) {
	r := receiptFixture("p", "/x.yaml", nil, nil)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	CompareReceipts(nil, r)
}
