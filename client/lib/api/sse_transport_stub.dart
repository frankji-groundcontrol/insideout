import 'dart:typed_data';

Stream<Uint8List> fetchSseStream(String url, Map<String, String> headers) =>
    throw UnsupportedError('fetch-based SSE is web-only');
