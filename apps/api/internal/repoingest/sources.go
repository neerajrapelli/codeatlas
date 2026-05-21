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
	resolver   CloneResolver
}

func NewGitSource(sourceType SourceType) *GitSource { return &GitSource{sourceType: sourceType} }

func (s *GitSource) SetCloneResolver(r CloneResolver) { s.resolver = r }

func (s *GitSource) Type() SourceType { return s.sourceType }

func (s *GitSource) Prepare(ctx context.Context, req CreateRequest, workspacePath string) error {
	if err := ValidateGitSourceURL(s.sourceType, req.SourceURL); err != nil {
		return err
	}
	if err := ValidateGitBranch(req.Branch); err != nil {
		return err
	}
	cloneURL := req.SourceURL
	if s.resolver != nil && req.ProviderTokenID != nil {
		authURL, err := s.resolver.ResolveCloneURL(ctx, req.TenantID, req.UserSubject, req.ProviderTokenID, s.sourceType, req.SourceURL)
		if err != nil {
			return err
		}
		cloneURL = authURL
	}
	cloneArgs := []string{"clone", "--depth", "1"}
	if req.Branch != "" {
		cloneArgs = append(cloneArgs, "--branch", req.Branch)
	}
	cloneArgs = append(cloneArgs, cloneURL, workspacePath)
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

// zipEntryTarget resolves a zip member path under workspaceRoot and rejects zip-slip paths.
func zipEntryTarget(workspaceRoot, name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("zip entry has empty name")
	}
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("zip path traversal detected: %s", name)
	}
	root := filepath.Clean(workspaceRoot)
	dest := filepath.Clean(filepath.Join(root, name))
	rel, err := filepath.Rel(root, dest)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("zip path traversal detected: %s", name)
	}
	return dest, nil
}

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
		mode := zf.Mode()
		if mode&os.ModeSymlink != 0 {
			return fmt.Errorf("zip symlink not allowed: %s", zf.Name)
		}
		if mode&(os.ModeDevice|os.ModeNamedPipe|os.ModeSocket) != 0 {
			return fmt.Errorf("zip special file not allowed: %s", zf.Name)
		}
		if zf.UncompressedSize64 > uint64(s.maxBytes) {
			return fmt.Errorf("zip entry exceeds max uncompressed size: %s", zf.Name)
		}

		clean, err := zipEntryTarget(workspacePath, zf.Name)
		if err != nil {
			return err
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
		n, copyErr := io.Copy(out, io.LimitReader(in, s.maxBytes+1))
		totalWritten += n
		_ = out.Close()
		_ = in.Close()
		if copyErr != nil {
			return fmt.Errorf("extract file: %w", copyErr)
		}
		if n > s.maxBytes {
			return fmt.Errorf("zip entry exceeds max size: %s", zf.Name)
		}
		if totalWritten > s.maxBytes*4 {
			return fmt.Errorf("extracted content exceeds safety limit")
		}
	}
	return nil
}
