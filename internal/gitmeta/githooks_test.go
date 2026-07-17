package gitmeta

import (
	"context"
	"testing"
)

func TestIsGitRepo_TrueForInitializedRepo(t *testing.T) {
	dir := initRepo(t, t.TempDir())

	if !IsGitRepo(context.Background(), dir) {
		t.Fatalf("IsGitRepo(%s) = false, want true", dir)
	}
}

func TestIsGitRepo_FalseForNonGitDir(t *testing.T) {
	dir := t.TempDir()

	if IsGitRepo(context.Background(), dir) {
		t.Fatalf("IsGitRepo(%s) = true, want false", dir)
	}
}
