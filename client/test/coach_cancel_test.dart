import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/sse.dart';

/// A live coach endpoint on loopback, shaped like the server's `POST
/// /conversations/{id}/messages`: it reads the request, then answers with one
/// delta frame and a ping comment every 50ms for as long as the client stays
/// connected. With [answer] false it takes the request and says nothing back,
/// like a turn that has not produced its first byte yet. [closed] completes
/// the moment the server sees the client leave.
class _CoachServer {
  _CoachServer._(this.http, this.requested, this.closed, this.requests);

  final HttpServer http;
  final Future<void> requested;
  final Future<void> closed;
  final List<String> requests;

  String get baseUrl => 'http://127.0.0.1:${http.port}/api/v1';

  static Future<_CoachServer> start({bool answer = true}) async {
    final requested = Completer<void>();
    final closed = Completer<void>();
    final requests = <String>[];
    final http = await HttpServer.bind('127.0.0.1', 0);
    http.listen((req) async {
      final body = await utf8.decoder.bind(req).join();
      requests.add('${req.method} ${req.uri.path} $body');
      if (!requested.isCompleted) requested.complete();
      // Unbuffered, so every write reaches the client as it is made, the way
      // the real handler flushes; a buffered dart:io response sends nothing.
      final res = req.response..bufferOutput = false;
      Timer? ping;
      if (answer) {
        res
          ..headers.contentType = ContentType('text', 'event-stream')
          ..write('event: message_start\ndata: {"id":"m1"}\n\n')
          ..write('event: delta\ndata: {"text":"Hel"}\n\n');
        ping = Timer.periodic(
          const Duration(milliseconds: 50),
          (_) => res.write(': ping\n\n'),
        );
      }
      // The response is done once its connection is: the client left.
      void gone() {
        ping?.cancel();
        if (!closed.isCompleted) closed.complete();
      }

      res.done.then((_) => gone(), onError: (_) => gone());
    });
    return _CoachServer._(http, requested.future, closed.future, requests);
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

/// Fails with [what] rather than a bare TimeoutException when [f] stays
/// pending: a stream that outlives its stop is the defect under test.
Future<T> _within<T>(Future<T> f, String what) =>
    f.timeout(_wait, onTimeout: () => throw TestFailure(what));

void main() {
  // readCoachSse winds down against the scheduler binding, as it does in the
  // app. The test binding stubs HTTP out, so hand real sockets back.
  TestWidgetsFlutterBinding.ensureInitialized();
  HttpOverrides.global = null;

  // A panel that disposes completes `stop`. The turn holds a model call open
  // for as long as it runs, so a connection left behind outlives the page by
  // the whole turn (or the 75s idle timeout) and the model keeps working for
  // a reader who is gone.
  test('completing stop ends a streaming coach turn and closes its '
      'connection', () async {
    final server = await _CoachServer.start();
    addTearDown(() => server.http.close(force: true));
    final firstDelta = Completer<void>();
    final stop = Completer<void>();

    final done = _api(server.baseUrl).sendCoach(
      'c1',
      'hello',
      CoachHandlers(
        onDelta: (_) {
          if (!firstDelta.isCompleted) firstDelta.complete();
        },
      ),
      stop: stop.future,
    );
    await firstDelta.future.timeout(_wait);

    stop.complete();

    // A stop is the caller leaving, not a failure: the call returns.
    await _within(done, 'sendCoach was still streaming 3s after stop');
    await _within(server.closed, 'the server never saw the client leave');
    expect(server.requests, [
      'POST /api/v1/conversations/c1/messages {"content":"hello"}',
    ]);
  });

  // The other half of the window: the POST is in, nothing has come back yet.
  test('completing stop before the server has answered closes the '
      'connection too', () async {
    final server = await _CoachServer.start(answer: false);
    addTearDown(() => server.http.close(force: true));
    final stop = Completer<void>();

    final done = _api(
      server.baseUrl,
    ).sendCoach('c1', 'hello', const CoachHandlers(), stop: stop.future);
    await server.requested.timeout(_wait);

    stop.complete();

    await _within(done, 'sendCoach was still waiting 3s after stop');
    await _within(server.closed, 'the server never saw the client leave');
  });
}
