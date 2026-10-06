import 'dart:typed_data';

import 'sse.dart';

Future<SseAttempt> fetchSseAttempt(
  String url,
  Map<String, String> headers, {
  String method = 'GET',
  String? body,
}) => throw UnsupportedError('fetch-based SSE is web-only');

Stream<Uint8List> fetchSseStream(
  String url,
  Map<String, String> headers, {
  String method = 'GET',
  String? body,
}) => throw UnsupportedError('fetch-based SSE is web-only');
