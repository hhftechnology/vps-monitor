package scanner

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
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
