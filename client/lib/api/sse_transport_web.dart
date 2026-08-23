import 'dart:js_interop';
import 'dart:typed_data';

import 'package:web/web.dart' as web;

import 'errors.dart';

// dio's web adapter is XHR + arraybuffer: the body only arrives when the
// connection closes, so a long-lived SSE stream never delivers frames.
// fetch + ReadableStream is the browser transport that actually streams.
Stream<Uint8List> fetchSseStream(String url, Map<String, String> headers) async* {
  final init = web.RequestInit(method: 'GET', headers: headers.jsify() as JSObject);
  final res = await web.window.fetch(url.toJS, init).toDart;
  if (!res.ok) {
    throw ApiException('stream failed', status: res.status);
  }
  final body = res.body;
  if (body == null) return;
  final reader = body.getReader() as web.ReadableStreamDefaultReader;
  while (true) {
    final chunk = await reader.read().toDart;
    if (chunk.done) break;
    final value = chunk.value;
    if (value == null) continue;
    yield (value as JSUint8Array).toDart;
  }
}
