package scanner

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// buildDockerLogStream constructs a multiplexed Docker log stream from a list
// of (streamType, payload) frames. streamType: 1 = stdout, 2 = stderr.
func buildDockerLogStream(frames []struct {
	stream byte
	data   []byte
}) []byte {
	var buf bytes.Buffer
	for _, f := range frames {
		header := make([]byte, 8)
		header[0] = f.stream
		binary.BigEndian.PutUint32(header[4:], uint32(len(f.data)))
		buf.Write(header)
		buf.Write(f.data)
	}
	return buf.Bytes()
}

func writeDockerLogFrame(w io.Writer, stream byte, data []byte) error {
	header := make([]byte, 8)
	header[0] = stream
	binary.BigEndian.PutUint32(header[4:], uint32(len(data)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

func TestStreamLogs_Demux(t *testing.T) {
	frames := []struct {
		stream byte
		data   []byte
	}{
		{1, []byte("hello ")},
		{2, []byte("warn1\n")},
		{1, []byte("world")},
		{2, []byte("warn2\n")},
		{1, []byte("!")},
	}
	stream := buildDockerLogStream(frames)

	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.txt")
	var n int64
	tail, err := streamLogs(bytes.NewReader(stream), outPath, &n)
	if err != nil {
		t.Fatalf("streamLogs: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if string(got) != "hello world!" {
		t.Errorf("stdout mismatch: got %q", got)
	}
	if atomic.LoadInt64(&n) != int64(len("hello world!")) {
		t.Errorf("byte count mismatch: %d", n)
	}
	if !strings.Contains(tail, "warn1") || !strings.Contains(tail, "warn2") {
		t.Errorf("stderr tail missing payloads: %q", tail)
	}
}

func TestParseGrypeOutputStream_Large(t *testing.T) {
	const N = 5000
	out := grypeOutput{Matches: make([]grypeMatch, N)}
	for i := 0; i < N; i++ {
		out.Matches[i] = grypeMatch{
			Vulnerability: grypeVulnerability{
				ID:       fmt.Sprintf("CVE-2024-%05d", i),
				Severity: "High",
				Fix:      grypeFixInfo{Versions: []string{"1.0.0"}},
			},
			Artifact: grypeArtifact{Name: "pkg", Version: "0.1"},
		}
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&out); err != nil {
		t.Fatalf("encode: %v", err)
	}
	vulns, err := parseGrypeOutputStream(&buf)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(vulns) != N {
		t.Errorf("got %d vulns, want %d", len(vulns), N)
	}
	if vulns[0].FixedVersion != "1.0.0" {
		t.Errorf("fix mismatch: %q", vulns[0].FixedVersion)
	}
}

func TestParseTrivyOutputStream_Large(t *testing.T) {
	const N = 3000
	out := trivyOutput{Results: []trivyResult{{Target: "img"}}}
	out.Results[0].Vulnerabilities = make([]trivyVulnerability, N)
	for i := 0; i < N; i++ {
		out.Results[0].Vulnerabilities[i] = trivyVulnerability{
			VulnerabilityID: fmt.Sprintf("CVE-2024-%05d", i),
			Severity:        "Critical",
			PkgName:         "pkg",
		}
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&out); err != nil {
		t.Fatalf("encode: %v", err)
	}
	vulns, err := parseTrivyOutputStream(&buf)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(vulns) != N {
		t.Errorf("got %d vulns, want %d", len(vulns), N)
	}
}

func TestStreamLogs_DelayedFrames(t *testing.T) {
	pr, pw := io.Pipe()
	outPath := filepath.Join(t.TempDir(), "delayed.txt")

	done := make(chan struct {
		tail string
		err  error
	}, 1)
	go func() {
		tail, err := streamLogs(pr, outPath, nil)
		done <- struct {
			tail string
			err  error
		}{tail: tail, err: err}
	}()

	go func() {
		defer pw.Close()
		_ = writeDockerLogFrame(pw, 1, []byte("hello"))
		time.Sleep(20 * time.Millisecond)
		_ = writeDockerLogFrame(pw, 2, []byte("warn\n"))
		time.Sleep(20 * time.Millisecond)
		_ = writeDockerLogFrame(pw, 1, []byte(" world"))
	}()

	result := <-done
	if result.err != nil {
		t.Fatalf("streamLogs: %v", result.err)
	}
	if !strings.Contains(result.tail, "warn") {
		t.Fatalf("stderr tail mismatch: %q", result.tail)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if string(got) != "hello world" {
		t.Fatalf("stdout mismatch: got %q", got)
	}
}

func TestScannerContainerLogsOptions(t *testing.T) {
	opts := scannerContainerLogsOptions()
	if !opts.ShowStdout || !opts.ShowStderr || !opts.Follow {
		t.Fatalf("unexpected logs options: %+v", opts)
	}
}

func TestEnrichParseErrorForEmptyOutputEOF(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(outPath, nil, 0o600); err != nil {
		t.Fatalf("write empty file: %v", err)
	}

	parseErr := fmt.Errorf("failed to parse grype output: %w", io.EOF)
	err := enrichParseErrorForEmptyOutput("grype", outPath, "line1\nline2", parseErr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "empty grype output") {
		t.Fatalf("expected empty output message, got %q", err)
	}
	if !strings.Contains(err.Error(), "stderr tail: line1 line2") {
		t.Fatalf("expected stderr tail in message, got %q", err)
	}
	if !strings.Contains(err.Error(), "EOF") {
		t.Fatalf("expected EOF in message, got %q", err)
	}
}

func TestEnrichParseErrorForEmptyOutput_SkipsNonEOFOrNonEmpty(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "non-empty.json")
	if err := os.WriteFile(outPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	parseErr := fmt.Errorf("failed to parse grype output: invalid character")
	if got := enrichParseErrorForEmptyOutput("grype", outPath, "stderr", parseErr); got != parseErr {
		t.Fatalf("expected same error for non-EOF parse error")
	}

	eofErr := fmt.Errorf("failed to parse grype output: %w", io.EOF)
	if got := enrichParseErrorForEmptyOutput("grype", outPath, "stderr", eofErr); got != eofErr {
		t.Fatalf("expected same error for non-empty output")
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	r := newRingBuffer(8)
	r.Write([]byte("abcd"))
	r.Write([]byte("efghij")) // total 10 -> last 8 = "cdefghij"
	if got := r.String(); got != "cdefghij" {
		t.Errorf("got %q, want %q", got, "cdefghij")
	}
	r.Write([]byte("KLMNOPQR")) // overwrite again
	if got := r.String(); got != "KLMNOPQR" {
		t.Errorf("got %q, want %q", got, "KLMNOPQR")
	}
}

// ─── BuildScannerHostConfig ───────────────────────────────────────────────────

func TestBuildScannerHostConfig_DockerSocketBind(t *testing.T) {
	hc := BuildScannerHostConfig(ScannerKindGrype, ScannerLimits{})
	found := false
	for _, b := range hc.Binds {
		if b == "/var/run/docker.sock:/var/run/docker.sock" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected docker socket bind in Binds, got %v", hc.Binds)
	}
}

func TestBuildScannerHostConfig_GrypeCacheVolumeBind(t *testing.T) {
	hc := BuildScannerHostConfig(ScannerKindGrype, ScannerLimits{})
	expected := ScannerCacheVolumes[ScannerKindGrype].Name + ":" + ScannerCacheVolumes[ScannerKindGrype].MountPath
	found := false
	for _, b := range hc.Binds {
		if b == expected {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected grype cache volume bind %q in %v", expected, hc.Binds)
	}
}

func TestBuildScannerHostConfig_TrivyCacheVolumeBind(t *testing.T) {
	hc := BuildScannerHostConfig(ScannerKindTrivy, ScannerLimits{})
	expected := ScannerCacheVolumes[ScannerKindTrivy].Name + ":" + ScannerCacheVolumes[ScannerKindTrivy].MountPath
	found := false
	for _, b := range hc.Binds {
		if b == expected {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected trivy cache volume bind %q in %v", expected, hc.Binds)
	}
}

func TestBuildScannerHostConfig_SyftCacheVolumeBind(t *testing.T) {
	hc := BuildScannerHostConfig(ScannerKindSyft, ScannerLimits{})
	expected := ScannerCacheVolumes[ScannerKindSyft].Name + ":" + ScannerCacheVolumes[ScannerKindSyft].MountPath
	found := false
	for _, b := range hc.Binds {
		if b == expected {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected syft cache volume bind %q in %v", expected, hc.Binds)
	}
}

func TestBuildScannerHostConfig_MemoryLimit(t *testing.T) {
	limits := ScannerLimits{MemoryBytes: 512 * 1024 * 1024}
	hc := BuildScannerHostConfig(ScannerKindGrype, limits)
	if hc.Resources.Memory != 512*1024*1024 {
		t.Fatalf("expected Memory=%d, got %d", 512*1024*1024, hc.Resources.Memory)
	}
}

func TestBuildScannerHostConfig_ZeroMemoryNotSet(t *testing.T) {
	hc := BuildScannerHostConfig(ScannerKindGrype, ScannerLimits{MemoryBytes: 0})
	if hc.Resources.Memory != 0 {
		t.Fatalf("expected Memory=0 when not set, got %d", hc.Resources.Memory)
	}
}

func TestBuildScannerHostConfig_PidsLimit(t *testing.T) {
	limits := ScannerLimits{PidsLimit: 256}
	hc := BuildScannerHostConfig(ScannerKindGrype, limits)
	if hc.Resources.PidsLimit == nil {
		t.Fatal("expected PidsLimit to be set, got nil")
	}
	if *hc.Resources.PidsLimit != 256 {
		t.Fatalf("expected PidsLimit=256, got %d", *hc.Resources.PidsLimit)
	}
}

func TestBuildScannerHostConfig_ZeroPidsNotSet(t *testing.T) {
	hc := BuildScannerHostConfig(ScannerKindGrype, ScannerLimits{PidsLimit: 0})
	if hc.Resources.PidsLimit != nil {
		t.Fatalf("expected PidsLimit=nil when not set, got %v", *hc.Resources.PidsLimit)
	}
}

func TestBuildScannerHostConfig_UnknownKindNoExtraBinds(t *testing.T) {
	// An unrecognised kind should not panic and should still include the socket.
	hc := BuildScannerHostConfig("unknown-kind", ScannerLimits{})
	if len(hc.Binds) != 1 || hc.Binds[0] != "/var/run/docker.sock:/var/run/docker.sock" {
		t.Fatalf("unexpected binds for unknown kind: %v", hc.Binds)
	}
}

// ─── TempScanFile ─────────────────────────────────────────────────────────────

func TestTempScanFile_ContainsJobID(t *testing.T) {
	path := TempScanFile("abc-123")
	if !strings.Contains(path, "abc-123") {
		t.Fatalf("expected job ID in path, got %q", path)
	}
}

func TestTempScanFile_HasJSONSuffix(t *testing.T) {
	path := TempScanFile("job-xyz")
	if !strings.HasSuffix(path, ".json") {
		t.Fatalf("expected .json suffix, got %q", path)
	}
}

func TestTempScanFile_DifferentIDsDifferentPaths(t *testing.T) {
	p1 := TempScanFile("id-1")
	p2 := TempScanFile("id-2")
	if p1 == p2 {
		t.Fatalf("different job IDs must produce different paths")
	}
}

// ─── readFilePrefix ───────────────────────────────────────────────────────────

func TestReadFilePrefix_ReadsUpToN(t *testing.T) {
	f := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(f, []byte("hello world"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := readFilePrefix(f, 5)
	if got != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}
}

func TestReadFilePrefix_ReturnsEmptyForMissingFile(t *testing.T) {
	got := readFilePrefix("/nonexistent/path/file.txt", 100)
	if got != "" {
		t.Fatalf("expected empty string for missing file, got %q", got)
	}
}

func TestReadFilePrefix_ReturnsFullContentWhenShorterThanN(t *testing.T) {
	f := filepath.Join(t.TempDir(), "short.txt")
	content := "hi"
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := readFilePrefix(f, 1024)
	if got != content {
		t.Fatalf("expected full content %q, got %q", content, got)
	}
}

func TestReadFilePrefix_TrimsWhitespace(t *testing.T) {
	f := filepath.Join(t.TempDir(), "ws.txt")
	if err := os.WriteFile(f, []byte("  hello  "), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := readFilePrefix(f, 100)
	if got != "hello" {
		t.Fatalf("expected trimmed content, got %q", got)
	}
}

// ─── countingWriter ───────────────────────────────────────────────────────────

func TestCountingWriter_CountsBytes(t *testing.T) {
	var buf strings.Builder
	var n int64
	cw := &countingWriter{w: &buf, n: &n}
	cw.Write([]byte("hello"))
	cw.Write([]byte(" world"))
	if n != 11 {
		t.Fatalf("expected n=11, got %d", n)
	}
	if buf.String() != "hello world" {
		t.Fatalf("unexpected content: %q", buf.String())
	}
}

func TestCountingWriter_NilCounter(t *testing.T) {
	// Must not panic when n is nil.
	var buf strings.Builder
	cw := &countingWriter{w: &buf, n: nil}
	if _, err := cw.Write([]byte("test")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── ScannerCacheVolumes constants ───────────────────────────────────────────

func TestScannerCacheVolumes_AllKindsDefined(t *testing.T) {
	for _, kind := range []ScannerKind{ScannerKindGrype, ScannerKindTrivy, ScannerKindSyft} {
		v, ok := ScannerCacheVolumes[kind]
		if !ok {
			t.Errorf("missing cache volume entry for kind %q", kind)
			continue
		}
		if v.Name == "" {
			t.Errorf("empty Name for kind %q", kind)
		}
		if v.MountPath == "" {
			t.Errorf("empty MountPath for kind %q", kind)
		}
	}
}

// ─── parseGrypeOutputStream edge cases ───────────────────────────────────────

func TestParseGrypeOutputStream_Empty(t *testing.T) {
	out := grypeOutput{Matches: []grypeMatch{}}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&out); err != nil {
		t.Fatalf("encode: %v", err)
	}
	vulns, err := parseGrypeOutputStream(&buf)
	if err != nil {
		t.Fatalf("unexpected error for empty matches: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("expected 0 vulns, got %d", len(vulns))
	}
}

func TestParseGrypeOutputStream_InvalidJSON(t *testing.T) {
	_, err := parseGrypeOutputStream(strings.NewReader("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseGrypeOutputStream_NormalizeSeverity(t *testing.T) {
	out := grypeOutput{Matches: []grypeMatch{
		{
			Vulnerability: grypeVulnerability{ID: "CVE-1", Severity: "critical"},
			Artifact:      grypeArtifact{Name: "pkg", Version: "1.0"},
		},
		{
			Vulnerability: grypeVulnerability{ID: "CVE-2", Severity: "UNKNOWN_SEVERITY"},
			Artifact:      grypeArtifact{Name: "pkg2", Version: "2.0"},
		},
	}}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(&out)

	vulns, err := parseGrypeOutputStream(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vulns[0].Severity != "Critical" {
		t.Errorf("expected Critical, got %q", vulns[0].Severity)
	}
	if vulns[1].Severity != "Unknown" {
		t.Errorf("expected Unknown for unrecognised severity, got %q", vulns[1].Severity)
	}
}

func TestParseGrypeOutputStream_MultipleFixVersions(t *testing.T) {
	// When there are multiple fix versions, the first one should be used.
	out := grypeOutput{Matches: []grypeMatch{
		{
			Vulnerability: grypeVulnerability{
				ID:       "CVE-2024-0001",
				Severity: "High",
				Fix:      grypeFixInfo{Versions: []string{"2.0.0", "3.0.0"}},
			},
			Artifact: grypeArtifact{Name: "lib", Version: "1.0"},
		},
	}}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(&out)

	vulns, err := parseGrypeOutputStream(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vulns[0].FixedVersion != "2.0.0" {
		t.Errorf("expected first fix version, got %q", vulns[0].FixedVersion)
	}
}

func TestParseGrypeOutputStream_NoFixVersions(t *testing.T) {
	out := grypeOutput{Matches: []grypeMatch{
		{
			Vulnerability: grypeVulnerability{ID: "CVE-2024-0002", Severity: "Low"},
			Artifact:      grypeArtifact{Name: "pkg", Version: "0.5"},
		},
	}}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(&out)

	vulns, err := parseGrypeOutputStream(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vulns[0].FixedVersion != "" {
		t.Errorf("expected empty FixedVersion when no fix, got %q", vulns[0].FixedVersion)
	}
}

// ─── parseTrivyOutputStream edge cases ───────────────────────────────────────

func TestParseTrivyOutputStream_Empty(t *testing.T) {
	out := trivyOutput{Results: []trivyResult{}}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(&out)

	vulns, err := parseTrivyOutputStream(&buf)
	if err != nil {
		t.Fatalf("unexpected error for empty results: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("expected 0 vulns, got %d", len(vulns))
	}
}

func TestParseTrivyOutputStream_InvalidJSON(t *testing.T) {
	_, err := parseTrivyOutputStream(strings.NewReader("{bad json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseTrivyOutputStream_MultipleResults(t *testing.T) {
	out := trivyOutput{Results: []trivyResult{
		{Target: "image1", Vulnerabilities: []trivyVulnerability{
			{VulnerabilityID: "CVE-1", Severity: "High", PkgName: "pkg1", InstalledVersion: "1.0"},
		}},
		{Target: "image2", Vulnerabilities: []trivyVulnerability{
			{VulnerabilityID: "CVE-2", Severity: "Critical", PkgName: "pkg2", InstalledVersion: "2.0"},
		}},
	}}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(&out)

	vulns, err := parseTrivyOutputStream(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vulns) != 2 {
		t.Fatalf("expected 2 vulns from 2 results, got %d", len(vulns))
	}
}

func TestParseTrivyOutputStream_ResultWithNoVulnerabilities(t *testing.T) {
	out := trivyOutput{Results: []trivyResult{
		{Target: "clean-image", Vulnerabilities: nil},
	}}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(&out)

	vulns, err := parseTrivyOutputStream(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vulns) != 0 {
		t.Fatalf("expected 0 vulns for clean image, got %d", len(vulns))
	}
}

func TestParseTrivyOutputStream_NormalizeSeverity(t *testing.T) {
	out := trivyOutput{Results: []trivyResult{
		{Target: "img", Vulnerabilities: []trivyVulnerability{
			{VulnerabilityID: "CVE-A", Severity: "MEDIUM", PkgName: "pkg"},
		}},
	}}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(&out)

	vulns, err := parseTrivyOutputStream(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vulns[0].Severity != "Medium" {
		t.Errorf("expected Medium, got %q", vulns[0].Severity)
	}
}

// ─── enrichParseErrorForEmptyOutput additional edge cases ─────────────────────

func TestEnrichParseErrorForEmptyOutput_LongStderrTruncatedTo512(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(outPath, nil, 0o600); err != nil {
		t.Fatalf("write empty file: %v", err)
	}

	longStderr := strings.Repeat("x", 1000)
	parseErr := fmt.Errorf("wrap: %w", io.EOF)
	err := enrichParseErrorForEmptyOutput("trivy", outPath, longStderr, parseErr)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	// The tail embedded in the error message must not exceed 512 chars of x's
	msg := err.Error()
	// Count 'x' characters — should be ≤ 512
	count := strings.Count(msg, "x")
	if count > 512 {
		t.Fatalf("stderr tail in error message too long: %d chars of 'x'", count)
	}
}

func TestEnrichParseErrorForEmptyOutput_EmptyStderrTail(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "empty2.json")
	if err := os.WriteFile(outPath, nil, 0o600); err != nil {
		t.Fatalf("write empty file: %v", err)
	}

	parseErr := fmt.Errorf("wrap: %w", io.EOF)
	err := enrichParseErrorForEmptyOutput("grype", outPath, "", parseErr)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if strings.Contains(err.Error(), "stderr tail") {
		t.Fatalf("empty stderr tail should not appear in message, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "empty grype output") {
		t.Fatalf("expected 'empty grype output' in message, got %q", err.Error())
	}
}