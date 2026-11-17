package uiserver

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

func linkIndexOutside(t *testing.T) (dir, outside string) {
	t.Helper()
	dir = copyGofixture(t)
	outside = t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, ".codegraph")); err != nil {
		t.Fatal(err)
	}
	return dir, outside
}

func TestGetStatusReportsIndexOutsideRoot(t *testing.T) {
	dir, outside := linkIndexOutside(t)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	_, err := client.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{}))
	if code := connect.CodeOf(err); code != connect.CodeFailedPrecondition {
		t.Fatalf("GetStatus code = %v (err %v), want CodeFailedPrecondition", code, err)
	}
	var cerr *connect.Error
	if !errors.As(err, &cerr) || cerr.Message() != indexOutsideRootMessage {
		t.Fatalf("GetStatus message = %v, want %q", err, indexOutsideRootMessage)
	}
	if strings.Contains(err.Error(), outside) {
		t.Fatalf("GetStatus error %q puts a host path on the wire", err)
	}
}

func TestComputeChangeDoesNotPublishNotInitializedForIndexOutsideRoot(t *testing.T) {
	dir, _ := linkIndexOutside(t)

	if _, event, changed := computeChange(context.Background(), dir, liveSignature{}, false); changed {
		t.Fatalf("computeChange published %+v for an index that leads outside the repository", event)
	}
}
