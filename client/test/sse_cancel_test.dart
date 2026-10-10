import 'dart:async';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';

/// A live SSE endpoint on loopback, shaped like the server's: one frame, then
/// a ping comment every 50ms for as long as the client stays connected.
/// [closed] completes the moment the server sees the client leave.
class _LiveServer {
  _LiveServer._(this.http, this.closed, this.paths);

  final HttpServer http;
  final Future<void> closed;
  final List<String> paths;

  String get baseUrl => 'http://127.0.0.1:${http.port}/api/v1';

  static Future<_LiveServer> start(String frame) async {
    final closed = Completer<void>();
    final paths = <String>[];
    final http = await HttpServer.bind('127.0.0.1', 0);
    http.listen((req) {
      paths.add('${req.uri.path}?${req.uri.query}');
      // Unbuffered, so every write reaches the client as it is made, the way
      // the real handler flushes; a buffered dart:io response sends nothing.
      final res = req.response
        ..bufferOutput = false
        ..headers.contentType = ContentType('text', 'event-stream')
        ..write(frame);
      final ping = Timer.periodic(
        const Duration(milliseconds: 50),
        (_) => res.write(': ping\n\n'),
      );
      // The response is done once its connection is: the client left.
      void gone() {
        ping.cancel();
        if (!closed.isCompleted) closed.complete();
      }

      res.done.then((_) => gone(), onError: (_) => gone());
    });
    return _LiveServer._(http, closed.future, paths);
  }
}

ApiClient _api(String baseUrl) => ApiClient(
  baseUrl: baseUrl,
  readAccess: () async => null,
  readRefresh: () async => null,
  writeTokens: (a, r) async {},
  clearTokens: () async {},
);

const _wait = Duration(seconds: 3);

void main() {
  // A page that disposes completes `stop`. The request must end with it, or
  // every roadmap visit leaves two connections open (and the viewer on the
  // other sessions' canvas) until the tab closes.
  test(
    'completing stop ends the progress stream and closes its connection',
    () async {
      final server = await _LiveServer.start(
        'event: progress\ndata: {"now":[],"needsDeadline":[],"next":[],"doneCount":0}\n\n',
      );
      addTearDown(() => server.http.close(force: true));
      final firstFrame = Completer<void>();
      final stop = Completer<void>();

      final done = _api(server.baseUrl).progressStream('p1', (_) {
        if (!firstFrame.isCompleted) firstFrame.complete();
      }, stop: stop.future);
      await firstFrame.future.timeout(_wait);

      stop.complete();

      await done.timeout(_wait); // returns; a stop is not a failure
      await server.closed.timeout(_wait); // and the server saw the client leave
      expect(server.paths, ['/api/v1/projects/p1/roadmap/progress/stream?']);
    },
  );

  test(
    'completing stop ends the presence stream and closes its connection',
    () async {
      final server = await _LiveServer.start('event: presence\ndata: []\n\n');
      addTearDown(() => server.http.close(force: true));
      final firstFrame = Completer<void>();
      final stop = Completer<void>();

      final done = _api(server.baseUrl).presenceStream('p1', 'c1', (_) {
        if (!firstFrame.isCompleted) firstFrame.complete();
      }, stop: stop.future);
      await firstFrame.future.timeout(_wait);

      stop.complete();

      await done.timeout(_wait);
      await server.closed.timeout(_wait);
      expect(server.paths, ['/api/v1/projects/p1/presence/stream?client=c1']);
    },
  );
}
