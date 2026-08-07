package fetcher

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"javawrapper/internal/config"
)

// DefaultJetBrainsGitURL adalah URL Git HTTP endpoint untuk JetBrainsRuntime.
var DefaultJetBrainsGitURL = "https://github.com/JetBrains/JetBrainsRuntime.git/info/refs?service=git-upload-pack"

// reJBRTag mencocokkan tag dengan format jbr-release-X.Y.ZbBUILD atau jbr-release-XbBUILD
// Contoh: jbr-release-21.0.3b465.3  -> Versi 21.0.3, Build b465.3, Major 21
// Contoh: jbr-release-17.0.10b1087.23 -> Versi 17.0.10, Build b1087.23, Major 17
var reJBRTag = regexp.MustCompile(`refs/tags/(jbr-release-(\d+)(?:\.([0-9.]+))?b([0-9.]+))$`)

// FetchJetBrainsSources mengambil seluruh tag release dari GitHub via Git Smart HTTP Protocol
// tanpa menyentuh GitHub REST API.
func FetchJetBrainsSources(gitHTTPURL string, ltsVersions []string) ([]config.JDKSource, error) {
	if gitHTTPURL == "" {
		gitHTTPURL = DefaultJetBrainsGitURL
	}

	resp, err := http.Get(gitHTTPURL)
	if err != nil {
		return nil, fmt.Errorf("gagal fetch git tags dari GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("git http request gagal dengan status %s", resp.Status)
	}

	return ParseJetBrainsGitRefs(resp.Body, ltsVersions)
}

// ParseJetBrainsGitRefs memuat data dari reader dan mengekstrak JDKSource untuk semua tag.
func ParseJetBrainsGitRefs(r io.Reader, ltsVersions []string) ([]config.JDKSource, error) {
	if len(ltsVersions) == 0 {
		ltsVersions = []string{"8", "11", "17", "21"}
	}

	scanner := bufio.NewScanner(r)
	seenTag := make(map[string]bool)
	var sources []config.JDKSource

	for scanner.Scan() {
		line := scanner.Text()
		matches := reJBRTag.FindStringSubmatch(line)
		if len(matches) < 5 {
			continue
		}

		fullTag := matches[1]  // jbr-release-21.0.3b465.3
		major := matches[2]    // 21
		verMinor := matches[3] // 0.3
		buildNum := matches[4] // 465.3

		// Cegah duplikasi tag yang persis sama
		if seenTag[fullTag] {
			continue
		}
		seenTag[fullTag] = true

		fullVersion := major
		if verMinor != "" {
			fullVersion = fmt.Sprintf("%s.%s", major, verMinor)
		}

		// Deteksi OS dan Arch saat runtime
		currOS := config.CurrentOS()
		currArch := config.CurrentArch()

		// Format URL download standar JetBrains Runtime
		// Contoh: https://github.com/JetBrains/JetBrainsRuntime/releases/download/jbr-release-21.0.3b465.3/jbr_jcef-21.0.3-linux-x64-b465.3.tar.gz
		downloadURL := fmt.Sprintf(
			"https://github.com/JetBrains/JetBrainsRuntime/releases/download/%s/jbr_jcef-%s-%s-%s-b%s.tar.gz",
			fullTag, fullVersion, currOS, currArch, buildNum,
		)

		isLTS := false
		for _, lts := range ltsVersions {
			if lts == major {
				isLTS = true
				break
			}
		}

		buildStr := fmt.Sprintf("b%s", buildNum)

		sources = append(sources, config.JDKSource{
			Vendor:  "JetBrains",
			Name:    "JetBrains Runtime",
			Version: fullVersion,
			Build:   buildStr,
			LTS:     isLTS,
			URL:     downloadURL,
			OS:      currOS,
			Arch:    currArch,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error membaca git refs stream: %w", err)
	}

	return sources, nil
}
