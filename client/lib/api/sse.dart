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
