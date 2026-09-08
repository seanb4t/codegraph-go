// client.ts is the browser's typed Connect client factory for
// codegraph.ui.v1.UIService (D-19). The Go binary serves both this SPA and
// the RPC surface from one origin (Phase 1's uiserver mux, D-09), so the
// transport's baseUrl is the same-origin root — an absolute
// http://127.0.0.1:... URL would break the moment the ephemeral port
// changes (Phase 1 D-07 binds 127.0.0.1:0).
//
// D-08: the wire encoding is Connect's JSON format, not binary protobuf —
// keeping every RPC inspectable in devtools and reproducible from a shell
// is worth more than wire size on a loopback socket. useBinaryFormat is
// set explicitly to false below even though it is already the installed
// @connectrpc/connect-web@2.1.2 package's own default ("By default,
// connect-web clients use the JSON format." —
// node_modules/@connectrpc/connect-web/dist/esm/connect-transport.d.ts) —
// an explicit value is what keeps D-08 true across a future dependency
// bump that might change that default.
//
// D-05: no `_connect.ts` companion module exists here or under
// web/src/lib/gen/ — Connect-ES v2 removed `protoc-gen-connect-es` and
// folded service-schema generation into `protoc-gen-es` itself, so the
// generated web/src/lib/gen/ui_pb.ts already exports the UIService
// GenService schema this module imports below.
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { UIService } from "$lib/gen/ui_pb";

export const transport = createConnectTransport({
	baseUrl: "/",
	useBinaryFormat: false,
});

export const uiClient = createClient(UIService, transport);
