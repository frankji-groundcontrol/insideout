import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/sse.dart';

void main() {
  test('parseSseBuffer splits delta frames and keeps a partial tail', () {
    final first = parseSseBuffer('event: delta\ndata: {"text":"Hel"}\n\nevent: delta\ndata: {"text":"lo');
    expect(first.frames, hasLength(1));
    expect(first.frames.single.event, 'delta');
    expect(first.rest, 'event: delta\ndata: {"text":"lo');

    final second = parseSseBuffer('${first.rest}"}\n\n');
    expect(second.frames, hasLength(1));
    expect(second.frames.single.data, '{"text":"lo"}');
    expect(second.rest, isEmpty);
  });

  test('applyCoachFrame appends delta text', () {
    final buf = StringBuffer();
    applyCoachFrame(const SseFrame('delta', '{"text":"Hi"}'), CoachHandlers(onDelta: buf.write));
    applyCoachFrame(const SseFrame('delta', '{"text":"!"}'), CoachHandlers(onDelta: buf.write));
    expect(buf.toString(), 'Hi!');
  });

  test('applyCoachFrame dispatches prd_updated stage_changed fact_recorded', () {
    String? section;
    String? stage;
    Map<String, dynamic>? fact;
    applyCoachFrame(
      const SseFrame('prd_updated', '{"section":"goals"}'),
      CoachHandlers(onPrdUpdated: (s) => section = s),
    );
    applyCoachFrame(
      const SseFrame('stage_changed', '{"stage":"critique"}'),
      CoachHandlers(onStageChanged: (s) => stage = s),
    );
    applyCoachFrame(
      const SseFrame('fact_recorded', '{"key":"users","value":"founders"}'),
      CoachHandlers(onFactRecorded: (f) => fact = f),
    );
    expect(section, 'goals');
    expect(stage, 'critique');
    expect(fact?['key'], 'users');
    expect(fact?['value'], 'founders');
  });

  test('applyCoachFrame throws ApiException on error frames', () {
    expect(
      () => applyCoachFrame(
        const SseFrame('error', '{"error":"slow down","code":"APP_THROTTLE","retry_after_seconds":9}'),
        const CoachHandlers(),
      ),
      throwsA(
        isA<ApiException>()
            .having((e) => e.message, 'message', 'slow down')
            .having((e) => e.code, 'code', 'APP_THROTTLE')
            .having((e) => e.retryAfterSeconds, 'retry', 9),
      ),
    );
  });
}
