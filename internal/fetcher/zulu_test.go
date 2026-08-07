package fetcher_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"javawrapper/internal/fetcher"
)

const sampleZuluJSON = `[
  {
    "abi": "any",
    "arch": "x86",
    "archive_type": "tar.gz",
    "availability_type": "CA",
    "crac_supported": false,
    "distro_version": [21, 34, 19, 0],
    "download_url": "https://cdn.azul.com/zulu/bin/zulu21.34.19-ca-jdk21.0.3-linux_x64.tar.gz",
    "hw_bitness": 64,
    "java_package_features": ["jdk"],
    "java_package_type": "jdk",
    "java_version": [21, 0, 3],
    "latest": true,
    "lib_c_type": "glibc",
    "name": "zulu21.34.19-ca-jdk21.0.3-linux_x64.tar.gz",
    "openjdk_build_number": 12,
    "os": "linux",
    "package_uuid": "test-uuid-1",
    "product": "zulu",
    "release_status": "ga",
    "sha256_hash": "dummyhash",
    "size": 200000000,
    "support_term": "lts"
  },
  {
    "abi": "any",
    "arch": "x86",
    "archive_type": "tar.gz",
    "availability_type": "CA",
    "distro_version": [25, 0, 1, 0],
    "download_url": "https://cdn.azul.com/zulu/bin/zulu25.0.1-ca-jdk25.0.0-linux_x64.tar.gz",
    "hw_bitness": 64,
    "java_package_features": ["jdk"],
    "java_package_type": "jdk",
    "java_version": [25, 0, 0],
    "latest": true,
    "lib_c_type": "glibc",
    "name": "zulu25.0.1-ca-jdk25.0.0-linux_x64.tar.gz",
    "openjdk_build_number": 1,
    "os": "linux",
    "package_uuid": "test-uuid-2",
    "product": "zulu",
    "release_status": "ea",
    "support_term": "sts"
  }
]`

func TestFetchZuluSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleZuluJSON))
	}))
	defer server.Close()

	sources, err := fetcher.FetchZuluSources(server.URL, []string{"8", "11", "17", "21"})
	if err != nil {
		t.Fatalf("FetchZuluSources error: %v", err)
	}

	// Hanya 1 item yang berstatus 'ga' (EA diabaikan)
	if len(sources) != 1 {
		t.Fatalf("diharapkan 1 source berstatus 'ga', didapat %d", len(sources))
	}

	s := sources[0]
	if s.Vendor != "Zulu" {
		t.Errorf("s.Vendor = %s, want Zulu", s.Vendor)
	}
	if s.Version != "21.0.3" {
		t.Errorf("s.Version = %s, want 21.0.3", s.Version)
	}
	if s.Build != "zulu21.34.19" {
		t.Errorf("s.Build = %s, want zulu21.34.19", s.Build)
	}
	if !s.LTS {
		t.Error("s.LTS = false, want true")
	}
	if s.OS != "linux" || s.Arch != "x64" {
		t.Errorf("s.OS/Arch = %s/%s, want linux/x64", s.OS, s.Arch)
	}
}
