import 'dart:js_interop';
import 'dart:typed_data';

import 'package:web/web.dart' as web;

import 'errors.dart';
import 'sse.dart';

// dio's web adapter is XHR + arraybuffer: the body only arrives when the
// connection closes, so a long-lived SSE stream never delivers frames.
// fetch + ReadableStream is the browser transport that actually streams.

/// Opens the fetch eagerly and returns the HTTP status alongside the body
/// stream — the caller can react to a 401 (refresh + retry) before any
/// frame is consumed (USER-10).
Future<SseAttempt> fetchSseAttempt(
  String url,
  Map<String, String> headers, {
  String method = 'GET',
  String? body,
}) async {
  final init = web.RequestInit(
    method: method,
    headers: headers.jsify() as JSObject,
    body: body?.toJS,
  );
  final res = await web.window.fetch(url.toJS, init).toDart;
  final responseBody = res.body;
  return SseAttempt(
    status: res.status,
    chunks: responseBody == null
        ? const Stream<Uint8List>.empty()
        : _readChunks(responseBody),
  );
}

Stream<Uint8List> _readChunks(web.ReadableStream stream) async* {
  final reader = stream.getReader() as web.ReadableStreamDefaultReader;
  while (true) {
    final chunk = await reader.read().toDart;
    if (chunk.done) break;
    final value = chunk.value;
    if (value == null) continue;
    yield (value as JSUint8Array).toDart;
  }
}

/// Throwing convenience for callers that have no retry policy of their own.
Stream<Uint8List> fetchSseStream(
  String url,
  Map<String, String> headers, {
  String method = 'GET',
  String? body,
}) async* {
  final attempt = await fetchSseAttempt(
    url,
    headers,
    method: method,
    body: body,
  );
  if (attempt.status >= 400) {
    throw ApiException('stream failed', status: attempt.status);
  }
  // Every chunk off the web ReadableStream is a JSUint8List; the cast is
  // the type-system bridge, not a conversion.
  yield* attempt.chunks.cast<Uint8List>();
}
