package fetcher_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"javawrapper/internal/fetcher"
)

const sampleGitRefs = `00000000# service=git-upload-pack
00000052aa719f98f3da6028d3ae1daef4f99aa642b8c1fe refs/tags/jbr-release-25.0.3b480.61
0051424f1e33dd0695943195793fa6f4f9e359151915 refs/tags/jbr-release-25.0.3b496.62
005152b04970adc8250cd5ad2947252750bd2e61efd5 refs/tags/jbr-release-21.0.3b465.3
0052e9be55de87444871a1acfcb3a6d5344a8f7c3587 refs/tags/jbr-release-17.0.10b1087.23
0000`

func TestFetchJetBrainsSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleGitRefs))
	}))
	defer server.Close()

	sources, err := fetcher.FetchJetBrainsSources(server.URL)
	if err != nil {
		t.Fatalf("FetchJetBrainsSources error: %v", err)
	}

	if len(sources) != 3 {
		t.Fatalf("diharapkan 3 sources (v25, v21, v17), didapat %d", len(sources))
	}

	// Cek v25
	s25 := sources[0]
	if s25.Version != "25" || s25.Vendor != "JetBrains" || s25.LTS {
		t.Errorf("v25 unexpected: %+v", s25)
	}

	// Cek v21
	s21 := sources[1]
	if s21.Version != "21" || s21.Vendor != "JetBrains" || !s21.LTS {
		t.Errorf("v21 unexpected: %+v", s21)
	}
	expectedURL21 := "https://github.com/JetBrains/JetBrainsRuntime/releases/download/jbr-release-21.0.3b465.3/jbr_jcef-21.0.3-linux-x64-b465.3.tar.gz"
	if s21.URL != expectedURL21 {
		t.Errorf("v21 URL = %s, want %s", s21.URL, expectedURL21)
	}

	// Cek v17
	s17 := sources[2]
	if s17.Version != "17" || s17.Vendor != "JetBrains" || !s17.LTS {
		t.Errorf("v17 unexpected: %+v", s17)
	}
}
