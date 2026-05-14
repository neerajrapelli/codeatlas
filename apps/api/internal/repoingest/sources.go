package repoingest

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Source interface {
	Type() SourceType
	Prepare(ctx context.Context, req CreateRequest, workspacePath string) error
}

type GitSource struct {
	sourceType SourceType
}

func NewGitSource(sourceType SourceType) *GitSource { return &GitSource{sourceType: sourceType} }
func (s *GitSource) Type() SourceType               { return s.sourceType }

func (s *GitSource) Prepare(ctx context.Context, req CreateRequest, workspacePath string) error {
	cloneArgs := []string{"clone", "--depth", "1"}
	if req.Branch != "" {
		cloneArgs = append(cloneArgs, "--branch", req.Branch)
	}
	cloneArgs = append(cloneArgs, req.SourceURL, workspacePath)
	cmd := exec.CommandContext(ctx, "git", cloneArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

type ZIPSource struct {
	maxBytes int64
	maxFiles int
}

func NewZIPSource(maxBytes int64, maxFiles int) *ZIPSource {
	return &ZIPSource{maxBytes: maxBytes, maxFiles: maxFiles}
}
func (s *ZIPSource) Type() SourceType { return SourceZIP }

func (s *ZIPSource) Prepare(_ context.Context, req CreateRequest, workspacePath string) error {
	file, err := os.Open(req.ZIPPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat zip: %w", err)
	}
	if info.Size() > s.maxBytes {
		return fmt.Errorf("zip exceeds max size (%d bytes)", s.maxBytes)
	}

	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return fmt.Errorf("parse zip: %w", err)
	}
	if len(reader.File) > s.maxFiles {
		return fmt.Errorf("zip exceeds max file count (%d)", s.maxFiles)
	}

	totalWritten := int64(0)
	for _, zf := range reader.File {
		targetPath := filepath.Join(workspacePath, zf.Name)
		clean := filepath.Clean(targetPath)
		if !strings.HasPrefix(clean, filepath.Clean(workspacePath)+string(os.PathSeparator)) && clean != filepath.Clean(workspacePath) {
			return fmt.Errorf("zip path traversal detected: %s", zf.Name)
		}

		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(clean, 0o755); err != nil {
				return fmt.Errorf("mkdir from zip: %w", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
			return fmt.Errorf("mkdir parent: %w", err)
		}
		in, err := zf.Open()
		if err != nil {
			return fmt.Errorf("open zipped file: %w", err)
		}
		out, err := os.OpenFile(clean, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			in.Close()
			return fmt.Errorf("create extracted file: %w", err)
		}
		n, copyErr := io.Copy(out, in)
		totalWritten += n
		_ = out.Close()
		_ = in.Close()
		if copyErr != nil {
			return fmt.Errorf("extract file: %w", copyErr)
		}
		if totalWritten > s.maxBytes*4 {
			return fmt.Errorf("extracted content exceeds safety limit")
		}
	}
	return nil
}
