@TestOn('browser')
library;

import 'dart:async';
import 'dart:convert';
import 'dart:js_interop';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/sse_transport_web.dart';
import 'package:web/web.dart' as web;

// TEST-11: the browser transport under the coach's SSE stream. `flutter test`
// on the VM cannot load it (it is fetch + ReadableStream) and `flutter build
// web` only compiles it, so these run in Chrome:
//
//   flutter test --platform chrome test/web
//
// The default VM run is unaffected: its collector drops @TestOn('browser')
// suites (`flutter test test/web` on the VM finds no tests), and the chrome
// compiler mounts only test/ as a source root, so the suites cannot live
// anywhere else.
//
// The origin is a throwaway Dart HTTP server on the VM side of the run (hybrid
// code), so the browser makes real cross-origin requests to it.

/// One tiny SSE origin. It sends the test its port, then serves:
///   /stream   frame one, then holds the response open until /release is hit,
///             then frame two and close;
///   /echo     one frame describing the request it got (method, headers, body);
///   anything else answers 401.
const _origin = r'''
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:stream_channel/stream_channel.dart';

Future<void> hybridMain(StreamChannel channel) async {
  final server = await HttpServer.bind('127.0.0.1', 0);
  final release = Completer<void>();
  channel.sink.add(server.port);
  server.listen((req) async {
    final res = req.response;
    // dart:io holds small writes back until the response closes unless told not to.
    res.bufferOutput = false;
    res.headers
      ..set('access-control-allow-origin', '*')
      ..set('access-control-allow-headers', 'authorization, content-type, accept')
      ..set('access-control-allow-methods', 'GET, POST, OPTIONS');
    if (req.method == 'OPTIONS') {
      res.statusCode = 204;
    } else if (req.uri.path == '/stream') {
      res.headers.contentType = ContentType('text', 'event-stream');
      res.write('event: delta\ndata: {"text":"one"}\n\n');
      await res.flush();
      await release.future;
      res.write('event: delta\ndata: {"text":"two"}\n\n');
    } else if (req.uri.path == '/release') {
      if (!release.isCompleted) release.complete();
      res.write('ok');
    } else if (req.uri.path == '/echo') {
      final body = await utf8.decodeStream(req);
      res.headers.contentType = ContentType('text', 'event-stream');
      res.write('event: echo\ndata: ${jsonEncode({
        'method': req.method,
        'authorization': req.headers.value('authorization'),
        'contentType': req.headers.contentType?.mimeType,
        'body': body,
      })}\n\n');
    } else {
      res.statusCode = 401;
      res.headers.contentType = ContentType.json;
      res.write('{"error":"nope"}');
    }
    await res.close();
  });
}
''';

Future<String> _startOrigin() async {
  final channel = spawnHybridCode(_origin);
  addTearDown(() => channel.sink.close());
  final port = await channel.stream.first as int;
  return 'http://127.0.0.1:$port';
}

const _sse = {'Accept': 'text/event-stream'};
const _wait = Duration(seconds: 5);

/// The next chunk off [chunks] as text, or null once the stream has ended.
Future<String?> _next(StreamIterator<Uint8List> chunks, String what) async {
  final more = await chunks.moveNext().timeout(
    _wait,
    onTimeout: () => fail('no chunk within ${_wait.inSeconds}s: $what'),
  );
  return more ? utf8.decode(chunks.current) : null;
}

void main() {
  test('hands over a frame while the response is still open', () async {
    final origin = await _startOrigin();
    final chunks = StreamIterator(fetchSseStream('$origin/stream', _sse));
    // Not awaited: a read the origin is still holding open would stall it.
    addTearDown(() => unawaited(chunks.cancel()));

    // The origin keeps frame two back until /release is hit, so frame one can
    // only arrive if the transport hands frames over as it reads them. One that
    // waits for the body to end (dio's XHR adapter) never gets past here.
    expect(
      await _next(chunks, 'frame one while the response is open'),
      contains('"text":"one"'),
    );

    await web.window.fetch('$origin/release'.toJS).toDart;
    expect(
      await _next(chunks, 'frame two after the release'),
      contains('"text":"two"'),
    );
    expect(await _next(chunks, 'the end of the stream'), isNull);
  });

  test('sends the method, headers and body it was given', () async {
    final origin = await _startOrigin();
    final bytes = await fetchSseStream(
      '$origin/echo',
      const {
        ..._sse,
        'Authorization': 'Bearer tok',
        'Content-Type': 'application/json',
      },
      method: 'POST',
      body: '{"content":"hi"}',
    ).expand((chunk) => chunk).toList().timeout(_wait);

    final data = RegExp(r'data: (.*)\n').firstMatch(utf8.decode(bytes))!;
    expect(jsonDecode(data.group(1)!), {
      'method': 'POST',
      'authorization': 'Bearer tok',
      'contentType': 'application/json',
      'body': '{"content":"hi"}',
    });
  });

  test(
    'a 401 throws from fetchSseStream and is reported by fetchSseAttempt',
    () async {
      final origin = await _startOrigin();

      await expectLater(
        fetchSseStream('$origin/denied', _sse).drain<void>(),
        throwsA(isA<ApiException>().having((e) => e.status, 'status', 401)),
      );

      final attempt = await fetchSseAttempt('$origin/denied', _sse);
      expect(attempt.status, 401);
      await attempt.chunks.drain<void>();
    },
  );
}
