package app

import "testing"

func FuzzDecodeCursor(f *testing.F) {
	f.Add("")
	f.Add(encodeCursor(0))
	f.Add(encodeCursor(25))
	f.Add("%%%")
	f.Add("bm90LWEtbnVtYmVy")
	f.Add("LTE=")

	f.Fuzz(func(t *testing.T, raw string) {
		got, err := decodeCursor(raw)
		if err == nil && got < 0 {
			t.Fatalf("decodeCursor(%q) returned negative offset %d", raw, got)
		}
	})
}

func FuzzParseResourceURI(f *testing.F) {
	seeds := []string{
		"linstor://clusters/homelab",
		"linstor://satellite-configurations/homelab",
		"linstor://node-connections/node-01",
		"linstor://nodes/node-01",
		"linstor://storage-pools/node-01/lvm-thick",
		"linstor://resource-definitions/pvc-123",
		"linstor://resources/pvc-123@node-01",
		"linstor://jobs/job_1",
		"linstor://bad",
		"",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, uri string) {
		kind, id, err := parseResourceURI(uri)
		if err != nil {
			return
		}
		if kind == "" || id == "" {
			t.Fatalf("parseResourceURI(%q) returned empty values: kind=%q id=%q", uri, kind, id)
		}
	})
}
