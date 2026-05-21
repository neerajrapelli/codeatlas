package repoingest

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestZip(t *testing.T, build func(w *zip.Writer)) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	build(w)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func TestZIPSource_rejectsPathTraversal(t *testing.T) {
	zipPath := writeTestZip(t, func(w *zip.Writer) {
		hdr := &zip.FileHeader{Name: "../../outside.txt"}
		hdr.Method = zip.Store
		wr, err := w.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = wr.Write([]byte("pwned"))
	})

	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}

	src := NewZIPSource(1<<20, 100)
	err := src.Prepare(context.Background(), CreateRequest{ZIPPath: zipPath}, workspace)
	if err == nil || !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("expected path traversal error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(workspace), "outside.txt")); statErr == nil {
		t.Fatal("traversal file was written outside workspace")
	}
}

func TestZIPSource_rejectsAbsoluteEntry(t *testing.T) {
	absName := "/etc/passwd"
	if filepath.Separator == '\\' {
		absName = `C:\outside.txt`
	}
	zipPath := writeTestZip(t, func(w *zip.Writer) {
		hdr := &zip.FileHeader{Name: absName}
		hdr.Method = zip.Store
		wr, err := w.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = wr.Write([]byte("secret"))
	})

	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}

	src := NewZIPSource(1<<20, 100)
	err := src.Prepare(context.Background(), CreateRequest{ZIPPath: zipPath}, workspace)
	if err == nil || !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("expected absolute path rejected, got %v", err)
	}
}

func TestZIPSource_rejectsSymlinkEntry(t *testing.T) {
	zipPath := writeTestZip(t, func(w *zip.Writer) {
		hdr := &zip.FileHeader{Name: "link"}
		hdr.SetMode(os.ModeSymlink | 0o755)
		if _, err := w.CreateHeader(hdr); err != nil {
			t.Fatal(err)
		}
	})

	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}

	src := NewZIPSource(1<<20, 100)
	err := src.Prepare(context.Background(), CreateRequest{ZIPPath: zipPath}, workspace)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejected, got %v", err)
	}
}

func TestZIPSource_extractsSafePaths(t *testing.T) {
	zipPath := writeTestZip(t, func(w *zip.Writer) {
		hdr := &zip.FileHeader{Name: "src/main.go"}
		hdr.Method = zip.Store
		wr, err := w.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = wr.Write([]byte("package main\n"))
	})

	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}

	src := NewZIPSource(1<<20, 100)
	if err := src.Prepare(context.Background(), CreateRequest{ZIPPath: zipPath}, workspace); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(workspace, "src", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "package main\n" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestZIPSource_rejectsTooManyFiles(t *testing.T) {
	zipPath := writeTestZip(t, func(w *zip.Writer) {
		for i := 0; i < 3; i++ {
			hdr := &zip.FileHeader{Name: filepath.Join("files", string(rune('a'+i))+".txt")}
			hdr.Method = zip.Store
			if _, err := w.CreateHeader(hdr); err != nil {
				t.Fatal(err)
			}
		}
	})

	workspace := filepath.Join(t.TempDir(), "workspace")
	src := NewZIPSource(1<<20, 2)
	err := src.Prepare(context.Background(), CreateRequest{ZIPPath: zipPath}, workspace)
	if err == nil || !strings.Contains(err.Error(), "max file count") {
		t.Fatalf("expected max file count error, got %v", err)
	}
}

func TestZipEntryTarget_rejectsDotDot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	_, err := zipEntryTarget(root, "../escape.txt")
	if err == nil || !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}
