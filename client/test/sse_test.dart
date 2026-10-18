import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/sse.dart';

void main() {
  test('coachSseOpen is POST JSON against /conversations/{id}/messages', () {
    final open = coachSseOpen(
      baseUrl: 'https://example.test/api/v1',
      conversationId: 'c1',
      content: 'hello',
      accessToken: 'tok',
    );
    expect(open.method, 'POST');
    expect(open.url, 'https://example.test/api/v1/conversations/c1/messages');
    expect(open.headers['Authorization'], 'Bearer tok');
    expect(open.headers['Accept'], 'text/event-stream');
    expect(open.headers['Content-Type'], 'application/json');
    expect(open.body, '{"content":"hello"}');
  });

  test('readCoachSse applies delta frames from a live chunk stream', () async {
    final buf = StringBuffer();
    await readCoachSse(
      Stream.fromIterable([
        utf8.encode('event: delta\ndata: {"text":"Hel"}\n\n'),
        utf8.encode('event: delta\ndata: {"text":"lo"}\n\n'),
      ]),
      CoachHandlers(onDelta: buf.write),
    );
    expect(buf.toString(), 'Hello');
  });

  test(
    'readCoachSse accepts Uint8List chunks from the web fetch transport',
    () async {
      final buf = StringBuffer();
      await readCoachSse(
        Stream<Uint8List>.fromIterable([
          Uint8List.fromList(
            utf8.encode('event: delta\ndata: {"text":"Hi"}\n\n'),
          ),
        ]),
        CoachHandlers(onDelta: buf.write),
      );
      expect(buf.toString(), 'Hi');
    },
  );

  test(
    'readProgressSse applies a progress frame as the assembled payload',
    () async {
      RoadmapProgress? got;
      await readProgressSse(
        Stream.fromIterable([
          utf8.encode(
            'event: progress\ndata: {"now":[{"id":"n1","title":"Ship it","status":"in_progress"}],"doneCount":0}\n\n',
          ),
        ]),
        (progress) => got = progress,
      );
      expect(got, isNotNull);
      expect(got!.now.single.title, 'Ship it');
      expect(got!.now.single.status, 'in_progress');
    },
  );

  test('parseSseBuffer splits delta frames and keeps a partial tail', () {
    final first = parseSseBuffer(
      'event: delta\ndata: {"text":"Hel"}\n\nevent: delta\ndata: {"text":"lo',
    );
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
    applyCoachFrame(
      const SseFrame('delta', '{"text":"Hi"}'),
      CoachHandlers(onDelta: buf.write),
    );
    applyCoachFrame(
      const SseFrame('delta', '{"text":"!"}'),
      CoachHandlers(onDelta: buf.write),
    );
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
    // The shape agent/sse.go factRecorded writes.
    applyCoachFrame(
      const SseFrame(
        'fact_recorded',
        '{"id":"f1","kind":"constraint","text":"Solo founders only","status":"proposed"}',
      ),
      CoachHandlers(onFactRecorded: (f) => fact = f),
    );
    expect(section, 'goals');
    expect(stage, 'critique');
    expect(fact?['kind'], 'constraint');
    expect(fact?['text'], 'Solo founders only');
  });

  test('applyCoachFrame throws ApiException on error frames', () {
    expect(
      () => applyCoachFrame(
        const SseFrame(
          'error',
          '{"error":"slow down","code":"APP_THROTTLE","retry_after_seconds":9}',
        ),
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

  group('coachSseWithRefresh (USER-10)', () {
    Future<(List<String>, Object?)> drain(Stream<List<int>> s) async {
      final out = <String>[];
      try {
        await for (final c in s) {
          out.add(utf8.decode(c));
        }
        return (out, null);
      } on Object catch (e) {
        return (out, e);
      }
    }

    test(
      '401 refreshes once and the retry carries the rotated token',
      () async {
        final log = <String>[];
        var n = 0;
        Future<SseAttempt> open() async {
          final i = n++;
          log.add('open-$i');
          if (i == 0) {
            return const SseAttempt(status: 401, chunks: Stream.empty());
          }
          return SseAttempt(
            status: 200,
            chunks: Stream.value(utf8.encode('data: frame-ok\n\n')),
          );
        }

        var refreshed = 0;
        var cleared = false;
        final (out, err) = await drain(
          coachSseWithRefresh(
            open,
            () async => refreshed++,
            () async => cleared = true,
          ),
        );
        expect(err, isNull);
        expect(out.join(), contains('frame-ok'));
        expect(log, ['open-0', 'open-1']);
        expect(refreshed, 1);
        expect(cleared, isFalse);
      },
    );

    test('failed refresh clears tokens and surfaces stream failed', () async {
      var cleared = false;
      var n = 0;
      Future<SseAttempt> open() async {
        n++;
        return const SseAttempt(status: 401, chunks: Stream.empty());
      }

      final (_, err) = await drain(
        coachSseWithRefresh(
          open,
          () async => throw ApiException('refresh rejected', status: 401),
          () async => cleared = true,
        ),
      );
      expect(
        cleared,
        isTrue,
        reason: 'a dead refresh must land the user on the auth door',
      );
      expect(err, isA<ApiException>().having((e) => e.status, 'status', 401));
      expect(n, 1, reason: 'no retry after a failed refresh');
    });

    test('retry still 401 clears tokens — no infinite loop', () async {
      var refreshed = 0;
      var cleared = false;
      var n = 0;
      Future<SseAttempt> open() async {
        n++;
        return const SseAttempt(status: 401, chunks: Stream.empty());
      }

      final (_, err) = await drain(
        coachSseWithRefresh(
          open,
          () async => refreshed++,
          () async => cleared = true,
        ),
      );
      expect(refreshed, 1, reason: 'exactly one refresh and one retry');
      expect(n, 2);
      expect(cleared, isTrue);
      expect(err, isA<ApiException>());
    });

    test('non-401 failure passes through without refresh', () async {
      var refreshed = 0;
      Future<SseAttempt> open() async =>
          const SseAttempt(status: 503, chunks: Stream.empty());
      final (_, err) = await drain(
        coachSseWithRefresh(open, () async => refreshed++, () async {}),
      );
      expect(refreshed, 0);
      expect(err, isA<ApiException>().having((e) => e.status, 'status', 503));
    });

    test('200 streams straight through, nothing touched', () async {
      var refreshed = 0;
      Future<SseAttempt> open() async => SseAttempt(
        status: 200,
        chunks: Stream.value(utf8.encode('data: direct\n\n')),
      );
      final (out, err) = await drain(
        coachSseWithRefresh(open, () async => refreshed++, () async {}),
      );
      expect(err, isNull);
      expect(out.join(), contains('direct'));
      expect(refreshed, 0);
    });
  });
}
