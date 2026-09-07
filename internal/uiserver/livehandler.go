package uiserver

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
)

// WatchGraph is plan 06-01's placeholder body for the fourteenth rpc —
// the service's first streaming method (RPC-04). Its only job in this
// wave is to satisfy the regenerated uiv1connect.UIServiceHandler
// interface so the package compiles; it builds no subscriber machinery
// and returns connect.CodeUnimplemented, named explicitly, because a
// generic internal error would satisfy compilation just as well and
// would then be indistinguishable from a real failure if it ever escaped
// this wave.
//
// 06-04 replaces this body with the real subscriber-register /
// Send-loop / deregister lifecycle. 06-04's own end-to-end test — a real
// Connect client receiving a real, watcher-triggered event over real
// HTTP/1.1 — cannot pass while this body survives, which is the
// structural gate that stops the placeholder from shipping.
func (s *uiService) WatchGraph(_ context.Context, _ *connect.Request[uiv1.WatchGraphRequest], _ *connect.ServerStream[uiv1.WatchGraphEvent]) error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("WatchGraph: not implemented until plan 06-04"))
}
