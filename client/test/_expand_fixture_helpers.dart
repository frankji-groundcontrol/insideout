import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:insideout/api/client.dart';

// rv4-FULL-01 test support. The two JSON files under
// server/internal/api/testdata/ are real responses of a running server (the
// draft is the 200 body of POST /roadmap/{id}/expand, the confirm is the 201
// body of POST /roadmap/{id}/expand/confirm). The Go test next to them,
// roadmap_expand_shape_test.go, re-checks on every run that the server still
// answers in these shapes; the Dart tests feed them to the real client.

String expandFixture(String name) =>
    File('../server/internal/api/testdata/$name').readAsStringSync();

/// The drafted subtasks of the captured draft, as the server wrote them.
List<Map<String, dynamic>> capturedDraftChildren() =>
    ((jsonDecode(expandFixture('roadmap_expand_draft.json'))
                as Map<String, dynamic>)['children']
            as List)
        .cast<Map<String, dynamic>>();

/// Stands in for the network under the real Dio stack: answers each
/// "METHOD /api/v1/path" with a canned status and JSON body and remembers
/// every request, so a test can say exactly what the client sent and nothing
/// more.
class ServerFixtures implements HttpClientAdapter {
  ServerFixtures(this.routes);

  final Map<String, (int, String)> routes;
  final requests = <RequestOptions>[];

  /// The sent calls as "METHOD /api/v1/path", in order.
  List<String> get calls => [
    for (final r in requests) '${r.method} ${r.uri.path}',
  ];

  /// The captured draft and confirm bodies for node [nodeId].
  factory ServerFixtures.captured(String nodeId) => ServerFixtures({
    'POST /api/v1/roadmap/$nodeId/expand': (
      200,
      expandFixture('roadmap_expand_draft.json'),
    ),
    'POST /api/v1/roadmap/$nodeId/expand/confirm': (
      201,
      expandFixture('roadmap_expand_confirm.json'),
    ),
  });

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    final (status, body) =
        routes['${options.method} ${options.uri.path}'] ??
        (404, '{"error":"no fixture for this call"}');
    return ResponseBody.fromString(
      body,
      status,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

/// The real [ApiClient] with [server] as its network.
ApiClient clientOver(ServerFixtures server) {
  final client = ApiClient(
    baseUrl: 'http://127.0.0.1:9/api/v1', // never dialed: the adapter answers
    readAccess: () async => null,
    readRefresh: () async => null,
    writeTokens: (a, r) async {},
    clearTokens: () async {},
  );
  client.dioForTests.httpClientAdapter = server;
  return client;
}
