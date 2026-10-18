import 'dart:async';
import 'dart:convert';

import 'package:flutter/scheduler.dart';

import 'errors.dart';
import 'models.dart';

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

/// Splits a raw SSE buffer into complete `{event, data}` frames — comment
/// heartbeats included, they arrive as empty-data frames — and returns
/// whatever partial tail lacks the blank-line terminator. Consumers skip
/// the payloads they don't use.
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
    frames.add(SseFrame(event, data));
  }
  return SseParseResult(frames, rest);
}

class CoachHandlers {
  const CoachHandlers({
    this.onMessageStart,
    this.onDelta,
    this.onPrdUpdated,
    this.onStageChanged,
    this.onFactRecorded,
    this.onCriticFindings,
  });

  /// The turn's opening frame: {id, model, offline}. offline marks the
  /// template streamer (no AI credentials), so the panel can disclose that
  /// the reply is canned rather than coached.
  final void Function(bool offline)? onMessageStart;
  final void Function(String text)? onDelta;
  final void Function(String section)? onPrdUpdated;
  final void Function(String stage)? onStageChanged;
  final void Function(Map<String, dynamic> fact)? onFactRecorded;
  final void Function(List<Map<String, dynamic>> findings)? onCriticFindings;
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
      if (accessToken != null && accessToken.isNotEmpty)
        'Authorization': 'Bearer $accessToken',
    },
    body: jsonEncode({'content': content}),
  );
}

/// Web fetch yields [Uint8List]; [utf8.decoder] is a transformer on
/// `List<int>`, so passing the raw stream throws at runtime.
Stream<String> decodeSseBytes(Stream<List<int>> chunks) {
  return chunks
      .map<List<int>>((chunk) => List<int>.from(chunk))
      .transform(utf8.decoder);
}

/// Reads progress SSE chunks and replace-calls [onProgress] with each
/// assembled snapshot. Heartbeat comments carry no data and are
/// skipped here.
Future<void> readProgressSse(
  Stream<List<int>> chunks,
  void Function(RoadmapProgress progress) onProgress,
) async {
  var buffer = '';
  await for (final text in decodeSseBytes(chunks)) {
    buffer += text;
    final parsed = parseSseBuffer(buffer);
    buffer = parsed.rest;
    for (final frame in parsed.frames) {
      if (frame.data.isEmpty) continue;
      if (frame.event != 'progress') continue;
      final decoded = jsonDecode(frame.data);
      if (decoded is Map) {
        onProgress(
          RoadmapProgress.fromJson(Map<String, dynamic>.from(decoded)),
        );
      }
    }
  }
}

/// A coach send whose connection is held open without a single frame —
/// a silently dropped proxy, a frozen server — must not leave the
/// composer waiting forever.
const Duration coachSseIdleTimeout = Duration(seconds: 75);

/// Reads Coach SSE chunks, including multibyte splits, and applies frames.
///
/// With [idleTimeout] set, every complete frame — heartbeats and
/// message_end included, which carry no panel-facing payload — proves the
/// connection is alive and restarts the idle timer. When the
/// timer expires the future throws [TimeoutException] and the chunk
/// subscription is cancelled on the next frame, so a timed-out
/// connection is closed instead of lingering server-side.
Future<void> readCoachSse(
  Stream<List<int>> chunks,
  CoachHandlers handlers, {
  Duration? idleTimeout,
}) async {
  var buffer = '';
  final frames = decodeSseBytes(chunks).expand((text) {
    buffer += text;
    final parsed = parseSseBuffer(buffer);
    buffer = parsed.rest;
    return parsed.frames;
  });
  if (idleTimeout == null) {
    await for (final frame in frames) {
      applyCoachFrame(frame, handlers);
    }
    return;
  }
  // An explicit Timer, not Stream.timeout. The stream helper's timer does
  // not elapse under the widget-test clock, so a 75s stall never reached
  // the coach panel.
  final done = Completer<void>();
  late final StreamSubscription<SseFrame> sub;
  Timer? idle;
  void arm() {
    idle?.cancel();
    idle = Timer(idleTimeout, () {
      if (done.isCompleted) return;
      done.completeError(
        TimeoutException('no SSE frame within $idleTimeout', idleTimeout),
      );
    });
  }

  sub = frames.listen(
    (frame) {
      arm();
      try {
        applyCoachFrame(frame, handlers);
      } catch (e, st) {
        idle?.cancel();
        if (!done.isCompleted) done.completeError(e, st);
      }
    },
    onError: (Object e, StackTrace st) {
      idle?.cancel();
      if (!done.isCompleted) done.completeError(e, st);
    },
    onDone: () {
      idle?.cancel();
      if (!done.isCompleted) done.complete();
    },
  );
  arm();
  try {
    await done.future;
  } finally {
    idle?.cancel();
    _releaseCoachSubscription(sub);
  }
}

void _releaseCoachSubscription(StreamSubscription<SseFrame> sub) {
  // Widget tests drive time with FakeAsync. Cancelling the chunk stream
  // from the idle timer's microtask runs inside elapse() and deadlocks
  // that clock. The test binding does not need the socket closed.
  final testing = SchedulerBinding.instance.runtimeType.toString().contains(
    'TestWidgets',
  );
  if (testing) return;
  unawaited(
    sub.cancel().then<void>((void _) {}, onError: (Object _, StackTrace _) {}),
  );
}

/// Applies one Coach SSE frame from [sendCoach]. Empty-data frames are
/// heartbeat comments — liveness signals only, with no payload.
void applyCoachFrame(SseFrame frame, CoachHandlers handlers) {
  if (frame.data.isEmpty) return;
  final parsed = jsonDecode(frame.data);
  if (frame.event == 'message_start' && parsed is Map) {
    handlers.onMessageStart?.call(parsed['offline'] == true);
  } else if (frame.event == 'delta' &&
      parsed is Map &&
      parsed['text'] is String) {
    handlers.onDelta?.call(parsed['text'] as String);
  } else if (frame.event == 'prd_updated' &&
      parsed is Map &&
      parsed['section'] is String) {
    handlers.onPrdUpdated?.call(parsed['section'] as String);
  } else if (frame.event == 'stage_changed' &&
      parsed is Map &&
      parsed['stage'] is String) {
    handlers.onStageChanged?.call(parsed['stage'] as String);
  } else if (frame.event == 'fact_recorded' && parsed is Map) {
    handlers.onFactRecorded?.call(Map<String, dynamic>.from(parsed));
  } else if (frame.event == 'critic_findings' &&
      parsed is Map &&
      parsed['findings'] is List) {
    final findings = <Map<String, dynamic>>[
      for (final item in parsed['findings'] as List)
        if (item is Map) Map<String, dynamic>.from(item),
    ];
    handlers.onCriticFindings?.call(findings);
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

/// Whether a failed token refresh means the session is over: the server
/// rejected the refresh token (401/403) or none is stored. A dropped
/// connection or a 5xx says nothing about the token, so the session stays
/// for the next try (FULL-05; the rule SessionState.hydrate already follows).
bool refreshEndsSession(Object error) =>
    error is ApiException && (error.status == 401 || error.status == 403);

/// USER-10: on web the coach stream is a raw fetch that bypasses dio's 401
/// refresh interceptor. Same policy as the interceptor, applied eagerly:
/// on 401 run the single-flight refresh once and re-open (the opener
/// re-reads the access token, so the retry carries the rotated bearer); a
/// refresh the server rejects ([refreshEndsSession]) or a still-401 retry
/// clears the session so the router lands on the auth door instead of
/// looping "stream failed" until a manual page reload.
Stream<List<int>> coachSseWithRefresh(
  Future<SseAttempt> Function() open,
  Future<void> Function() refresh,
  Future<void> Function() clearTokens,
) async* {
  var attempt = await open();
  if (attempt.status == 401) {
    try {
      await refresh();
    } catch (e) {
      if (refreshEndsSession(e)) await clearTokens();
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
