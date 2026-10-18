import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/sse.dart';

/// FULL-04: on web the presence and progress streams are raw fetches, which
/// bypass the dio interceptor that owns the 401 refresh. An expired access
/// token (the server ends a stream when its token expires, and the page
/// reconnects with the stored one) threw "stream failed" and the stream never
/// came back. They must follow the coach's policy: one single-flight refresh,
/// then reopen with the rotated bearer.
///
/// The fetch transport exists only in a browser, so this drives the web path
/// on the VM through the client's transport seam with scripted responses; it
/// is not a browser run.

class _World {
  _World(this.statuses, {this.refreshError});

  /// The status each attempt gets, in order; the last one repeats.
  final List<int> statuses;
  final Object? refreshError;

  String access = 'access-old';
  int refreshes = 0;
  bool cleared = false;
  final urls = <String>[];
  final bearers = <String?>[];

  Future<SseAttempt> fetch(
    String url,
    Map<String, String> headers, {
    Future<void>? stop,
  }) async {
    urls.add(url);
    bearers.add(headers['Authorization']);
    final status = statuses[(urls.length - 1).clamp(0, statuses.length - 1)];
    return SseAttempt(
      status: status,
      chunks: status == 200
          ? Stream<List<int>>.value(utf8.encode(_frame))
          : const Stream<List<int>>.empty(),
    );
  }

  // One frame the presence reader and the progress reader each take as theirs.
  static const _frame =
      'event: presence\ndata: [{"sessionId":"s1","userId":"u1","name":"Ada"}]\n\n'
      'event: progress\ndata: {"now":[],"needsDeadline":[],"next":[],"doneCount":3}\n\n';
}

class _Api extends ApiClient {
  _Api(_World world)
    : _world = world,
      super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => world.access,
        readRefresh: () async => 'refresh-1',
        writeTokens: (a, r) async => world.access = a,
        clearTokens: () async => world.cleared = true,
      ) {
    fetchAttemptForTests = world.fetch;
  }

  final _World _world;

  @override
  Future<void> refresh() async {
    _world.refreshes++;
    final error = _world.refreshError;
    if (error != null) throw error;
    _world.access = 'access-new';
  }
}

typedef _Open = Future<void> Function(_Api api, void Function() delivered);

final _streams = <String, _Open>{
  'presence': (api, delivered) => api.presenceStream('p1', 'tab-1', (entries) {
    expect(entries.single.name, 'Ada');
    delivered();
  }),
  'progress': (api, delivered) => api.progressStream('p1', (RoadmapProgress p) {
    expect(p.doneCount, 3);
    delivered();
  }),
};

Matcher _apiStatus(int status) =>
    throwsA(isA<ApiException>().having((e) => e.status, 'status', status));

void main() {
  for (final entry in _streams.entries) {
    group('${entry.key} stream', () {
      test(
        'a 401 refreshes once and reopens with the rotated bearer',
        () async {
          final world = _World([401, 200]);
          var frames = 0;

          await entry.value(_Api(world), () => frames++);

          expect(world.refreshes, 1);
          expect(world.bearers, ['Bearer access-old', 'Bearer access-new']);
          expect(frames, 1, reason: 'the reopened stream delivers its frames');
          expect(world.cleared, isFalse);
          expect(world.urls.last, contains('/projects/p1/'));
        },
      );

      test('a healthy stream never refreshes', () async {
        final world = _World([200]);
        var frames = 0;

        await entry.value(_Api(world), () => frames++);

        expect(world.refreshes, 0);
        expect(world.bearers, ['Bearer access-old']);
        expect(frames, 1);
      });

      test('a 401 that survives the refresh ends the session', () async {
        final world = _World([401]);

        await expectLater(entry.value(_Api(world), () {}), _apiStatus(401));

        expect(world.refreshes, 1);
        expect(world.urls, hasLength(2));
        expect(world.cleared, isTrue);
      });

      test('a refresh the server rejects ends the session', () async {
        final world = _World([
          401,
        ], refreshError: ApiException('no', status: 401));

        await expectLater(entry.value(_Api(world), () {}), _apiStatus(401));

        expect(world.urls, hasLength(1));
        expect(world.cleared, isTrue);
      });

      test('a failure that is not a 401 does not refresh', () async {
        final world = _World([503]);

        await expectLater(entry.value(_Api(world), () {}), _apiStatus(503));

        expect(world.refreshes, 0);
        expect(world.cleared, isFalse);
      });
    });
  }
}
