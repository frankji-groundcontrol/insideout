import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/sse.dart';

/// Stands in for the network under the real Dio stack: interceptors run,
/// options merge with the shared BaseOptions, and `RequestOptions
/// .receiveTimeout` is the value dio would enforce between response chunks —
/// the budget under test, seen without a live socket.
class _RecordingAdapter implements HttpClientAdapter {
  final requests = <RequestOptions>[];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    if (options.method == 'POST') {
      // One frame then end-of-stream, so readCoachSse completes cleanly.
      const body = 'event: message_start\ndata: {"id":"m1"}\n\n';
      return ResponseBody(
        Stream.value(Uint8List.fromList(utf8.encode(body))),
        200,
        headers: {
          Headers.contentTypeHeader: ['text/event-stream'],
        },
      );
    }
    return ResponseBody.fromString(
      jsonEncode({
        'id': 'user-1',
        'email': 'ada@example.com',
        'username': 'ada',
        'bio': '',
        'keywords': <String>[],
      }),
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

ApiClient _client(_RecordingAdapter recorder) {
  final client = ApiClient(
    baseUrl: 'http://localhost:9', // never dialed: the adapter is fake
    readAccess: () async => 'access-1',
    readRefresh: () async => 'refresh-1',
    writeTokens: (a, r) async {},
    clearTokens: () async {},
  );
  client.dioForTests.httpClientAdapter = recorder;
  return client;
}

void main() {
  // readCoachSse touches SchedulerBinding when releasing its subscription.
  TestWidgetsFlutterBinding.ensureInitialized();

  // The coach POST is one response whose body trickles for the whole model
  // turn. The shared 30s receiveTimeout fires between chunks, so a silent
  // provider (first delta on a large context, the server's tighten retry)
  // kills the native turn at 30s while the server tolerates 90s and
  // readCoachSse's own idle timer waits 75s. This request alone must turn
  // the timeout off — dio reads zero as no limit — so the 75s timer governs.
  test('the native coach POST turns the shared receiveTimeout off', () async {
    final recorder = _RecordingAdapter();
    await _client(recorder).sendCoach('c1', 'hello', const CoachHandlers());

    final coach = recorder.requests.singleWhere(
      (o) => o.path == '/conversations/c1/messages',
    );
    expect(coach.receiveTimeout, Duration.zero);
  });

  // Only the coach stream is exempt: every ordinary request keeps the
  // shared 30s ceiling, so a hung JSON endpoint still fails in bounded time.
  test('ordinary requests keep the shared 30s receiveTimeout', () async {
    final recorder = _RecordingAdapter();
    await _client(recorder).me();

    final me = recorder.requests.singleWhere((o) => o.path == '/me');
    expect(me.receiveTimeout, const Duration(seconds: 30));
  });
}
