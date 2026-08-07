package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"javawrapper/internal/config"
)

// DefaultZuluAPIURL adalah URL Metadata API resmi dari Azul Zulu.
var DefaultZuluAPIURL = "https://api.azul.com/metadata/v1/zulu/packages?availability_types=ca&release_status=both&page_size=1000&include_fields=java_package_features,release_status,support_term,os,arch,hw_bitness,abi,java_package_type,sha256_hash,size,archive_type,lib_c_type,crac_supported&page=1&azul_com=true"

// AzulPackage mendefinisikan struktur item dari API metadata Azul.
type AzulPackage struct {
	Name               string   `json:"name"`
	PackageUUID        string   `json:"package_uuid"`
	Product            string   `json:"product"`
	AvailabilityType   string   `json:"availability_type"`
	ReleaseStatus      string   `json:"release_status"`
	SupportTerm        string   `json:"support_term"`
	OS                 string   `json:"os"`
	Arch               string   `json:"arch"`
	HWBitness          int      `json:"hw_bitness"`
	ABI                string   `json:"abi"`
	JavaPackageType    string   `json:"java_package_type"`
	JavaPackageFeatures []string `json:"java_package_features"`
	SHA256Hash         string   `json:"sha256_hash"`
	Size               int64    `json:"size"`
	ArchiveType        string   `json:"archive_type"`
	LibCType           *string  `json:"lib_c_type"`
	DownloadURL        string   `json:"download_url"`
	JavaVersion        []int    `json:"java_version"`
	DistroVersion      []int    `json:"distro_version"`
	OpenJDKBuildNumber int      `json:"openjdk_build_number"`
	Latest             bool     `json:"latest"`
}

// FetchZuluSources mengambil metadata rilis JDK dari Azul API.
func FetchZuluSources(apiURL string, ltsVersions []string) ([]config.JDKSource, error) {
	if apiURL == "" {
		apiURL = DefaultZuluAPIURL
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("gagal HTTP request ke Azul API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Azul API mengembalikan status %s", resp.Status)
	}

	return ParseZuluPackages(resp.Body, ltsVersions)
}

// ParseZuluPackages mendecode JSON dari reader dan menyaring rilis JDK Zulu (GA, glibc, tar.gz/zip).
func ParseZuluPackages(r io.Reader, ltsVersions []string) ([]config.JDKSource, error) {
	var pkgs []AzulPackage
	if err := json.NewDecoder(r).Decode(&pkgs); err != nil {
		return nil, fmt.Errorf("gagal parse JSON Azul API: %w", err)
	}

	var sources []config.JDKSource
	seenTag := make(map[string]bool)

	for _, p := range pkgs {
		// Filter 1: Hanya rilis GA (General Availability / Release)
		if !strings.EqualFold(p.ReleaseStatus, "ga") {
			continue
		}

		// Filter 2: Hanya tipe JDK (bukan JRE)
		if !strings.EqualFold(p.JavaPackageType, "jdk") {
			continue
		}

		// Filter 3: Hanya format archive tar.gz atau zip (tergantung OS)
		archType := strings.ToLower(p.ArchiveType)
		if archType != "tar.gz" && archType != "zip" {
			continue
		}

		// Filter 4: Hanya versi glibc (atau nil untuk OS yang tidak memakai libc type khusus)
		if p.LibCType != nil && *p.LibCType != "" && !strings.EqualFold(*p.LibCType, "glibc") {
			continue
		}

		// Format versi Java: [21, 0, 3] -> "21.0.3", [21, 0, 0] -> "21"
		verStr := formatJavaVersion(p.JavaVersion)
		if verStr == "" {
			continue
		}

		// Format Build: distro_version [21, 34, 19, 0] -> "zulu21.34.19"
		buildStr := formatZuluBuild(p.DistroVersion, p.OpenJDKBuildNumber)

		// Unik identifier per rilis
		tagKey := fmt.Sprintf("%s-%s-%s-%s-%s", p.OS, p.Arch, verStr, buildStr, p.DownloadURL)
		if seenTag[tagKey] {
			continue
		}
		seenTag[tagKey] = true

		// Normalisasi OS & Arch
		normOS := config.NormalizeOS(p.OS)
		normArch := config.NormalizeArchWithBitness(p.Arch, p.HWBitness)

		// Tentukan LTS dari field support_term atau daftar ltsVersions
		isLTS := strings.EqualFold(p.SupportTerm, "lts")
		if !isLTS && len(p.JavaVersion) > 0 {
			majorStr := fmt.Sprintf("%d", p.JavaVersion[0])
			for _, lts := range ltsVersions {
				if lts == majorStr {
					isLTS = true
					break
				}
			}
		}

		sources = append(sources, config.JDKSource{
			Vendor:  "Zulu",
			Name:    "Zulu OpenJDK",
			Version: verStr,
			Build:   buildStr,
			LTS:     isLTS,
			URL:     p.DownloadURL,
			OS:      normOS,
			Arch:    normArch,
		})
	}

	return sources, nil
}

func formatJavaVersion(jv []int) string {
	if len(jv) == 0 {
		return ""
	}
	if len(jv) == 1 {
		return fmt.Sprintf("%d", jv[0])
	}
	if len(jv) == 2 {
		if jv[1] == 0 {
			return fmt.Sprintf("%d", jv[0])
		}
		return fmt.Sprintf("%d.%d", jv[0], jv[1])
	}
	// 3 atau lebih
	if jv[1] == 0 && jv[2] == 0 {
		return fmt.Sprintf("%d", jv[0])
	}
	return fmt.Sprintf("%d.%d.%d", jv[0], jv[1], jv[2])
}

func formatZuluBuild(dv []int, openjdkBuild int) string {
	if len(dv) >= 3 {
		return fmt.Sprintf("zulu%d.%d.%d", dv[0], dv[1], dv[2])
	}
	if openjdkBuild > 0 {
		return fmt.Sprintf("b%d", openjdkBuild)
	}
	return "-"
}
