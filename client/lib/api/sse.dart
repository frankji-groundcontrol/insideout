import 'dart:convert';

import 'errors.dart';

class SseFrame {
  const SseFrame(this.event, this.data);

  final String event;
  final String data;
}

class SseParseResult {
  const SseParseResult(this.frames, this.rest);

  final List<SseFrame> frames;
  final String rest;
}

/// Splits a raw SSE buffer into complete `{event, data}` frames.
SseParseResult parseSseBuffer(String buffer) {
  final frames = <SseFrame>[];
  var rest = buffer;
  while (true) {
    final sep = rest.indexOf('\n\n');
    if (sep < 0) break;
    final raw = rest.substring(0, sep);
    rest = rest.substring(sep + 2);
    var event = 'message';
    var data = '';
    for (final line in raw.split('\n')) {
      if (line.startsWith('event: ')) event = line.substring(7);
      if (line.startsWith('data: ')) data = line.substring(6);
    }
    if (data.isEmpty) continue;
    frames.add(SseFrame(event, data));
  }
  return SseParseResult(frames, rest);
}

class CoachHandlers {
  const CoachHandlers({
    this.onDelta,
    this.onPrdUpdated,
    this.onStageChanged,
    this.onFactRecorded,
  });

  final void Function(String text)? onDelta;
  final void Function(String section)? onPrdUpdated;
  final void Function(String stage)? onStageChanged;
  final void Function(Map<String, dynamic> fact)? onFactRecorded;
}

/// One Coach POST that must stream on web via fetch (BUG-014), not dio/XHR.
class CoachSseOpen {
  const CoachSseOpen({
    required this.method,
    required this.url,
    required this.headers,
    required this.body,
  });

  final String method;
  final String url;
  final Map<String, String> headers;
  final String body;
}

CoachSseOpen coachSseOpen({
  required String baseUrl,
  required String conversationId,
  required String content,
  String? accessToken,
}) {
  return CoachSseOpen(
    method: 'POST',
    url: '$baseUrl/conversations/$conversationId/messages',
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'text/event-stream',
      if (accessToken != null && accessToken.isNotEmpty) 'Authorization': 'Bearer $accessToken',
    },
    body: jsonEncode({'content': content}),
  );
}

/// Web fetch yields [Uint8List]; [utf8.decoder] is a transformer on
/// `List<int>`, so passing the raw stream throws at runtime.
Stream<String> decodeSseBytes(Stream<List<int>> chunks) {
  return chunks.map<List<int>>((chunk) => List<int>.from(chunk)).transform(utf8.decoder);
}

/// Reads Coach SSE chunks, including multibyte splits, and applies frames.
Future<void> readCoachSse(Stream<List<int>> chunks, CoachHandlers handlers) async {
  var buffer = '';
  await for (final text in decodeSseBytes(chunks)) {
    buffer += text;
    final parsed = parseSseBuffer(buffer);
    buffer = parsed.rest;
    for (final frame in parsed.frames) {
      applyCoachFrame(frame, handlers);
    }
  }
}

/// Applies one Coach SSE frame from [sendCoach].
void applyCoachFrame(SseFrame frame, CoachHandlers handlers) {
  final parsed = jsonDecode(frame.data);
  if (frame.event == 'delta' && parsed is Map && parsed['text'] is String) {
    handlers.onDelta?.call(parsed['text'] as String);
  } else if (frame.event == 'prd_updated' && parsed is Map && parsed['section'] is String) {
    handlers.onPrdUpdated?.call(parsed['section'] as String);
  } else if (frame.event == 'stage_changed' && parsed is Map && parsed['stage'] is String) {
    handlers.onStageChanged?.call(parsed['stage'] as String);
  } else if (frame.event == 'fact_recorded' && parsed is Map) {
    handlers.onFactRecorded?.call(Map<String, dynamic>.from(parsed));
  } else if (frame.event == 'error' && parsed is Map) {
    throw toApiError(null, Map<String, dynamic>.from(parsed));
  }
}

/// One eagerly-opened SSE response: the HTTP status is known before any
/// frame is read, which is what lets the coach's raw-fetch path see a 401
/// before consuming the stream (USER-10).
class SseAttempt {
  const SseAttempt({required this.status, required this.chunks});

  final int status;
  final Stream<List<int>> chunks;
}

/// USER-10: on web the coach stream is a raw fetch that bypasses dio's 401
/// refresh interceptor. Same policy as the interceptor, applied eagerly:
/// on 401 run the single-flight refresh once and re-open (the opener
/// re-reads the access token, so the retry carries the rotated bearer); a
/// failed refresh or a still-401 retry clears the session so the router
/// lands on the auth door instead of looping "stream failed" until a
/// manual page reload.
Stream<List<int>> coachSseWithRefresh(
  Future<SseAttempt> Function() open,
  Future<void> Function() refresh,
  Future<void> Function() clearTokens,
) async* {
  var attempt = await open();
  if (attempt.status == 401) {
    try {
      await refresh();
    } catch (_) {
      await clearTokens();
      throw ApiException('stream failed', status: attempt.status);
    }
    attempt = await open();
    if (attempt.status == 401) {
      await clearTokens();
      throw ApiException('stream failed', status: attempt.status);
    }
  }
  if (attempt.status >= 400) {
    throw ApiException('stream failed', status: attempt.status);
  }
  yield* attempt.chunks;
}
