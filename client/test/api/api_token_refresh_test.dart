import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';

/// Loopback HTTP fake so the real Dio interceptor stack runs: 401s, refresh
/// POSTs, and retried requests all cross real wire I/O.
class _FakeServer {
  _FakeServer._(this.http, this.hits);

  final HttpServer http;
  final List<String> hits;

  String get baseUrl => 'http://127.0.0.1:${http.port}/api/v1';

  int get hitsForMe => hits.where((h) => h == 'GET /api/v1/me').length;
  int get hitsForRefresh =>
      hits.where((h) => h == 'POST /api/v1/auth/refresh').length;

  static Future<_FakeServer> start({
    required bool refreshSucceeds,
    bool meAlways401 = false,
  }) async {
    final hits = <String>[];
    final http = await HttpServer.bind('127.0.0.1', 0);
    http.listen((req) async {
      await utf8.decoder.bind(req).join();
      hits.add('${req.method} ${req.uri.path}');
      final auth = req.headers.value(HttpHeaders.authorizationHeader);
      void json(int status, Object data) {
        req.response
          ..statusCode = status
          ..headers.contentType = ContentType.json
          ..write(jsonEncode(data));
      }

      switch (req.uri.path) {
        case '/api/v1/me':
          if (!meAlways401 && auth == 'Bearer access-2') {
            json(200, {
              'id': 'user-1',
              'email': 'ada@example.com',
              'username': 'ada',
              'bio': '',
              'keywords': <String>[],
            });
          } else {
            json(401, {'error': 'invalid or expired session'});
          }
        case '/api/v1/auth/refresh':
          if (refreshSucceeds) {
            json(200, {
              'ok': true,
              'accessToken': 'access-2',
              'refreshToken': 'refresh-2',
            });
          } else {
            json(401, {'error': 'invalid refresh token'});
          }
        default:
          json(404, {'error': 'not found'});
      }
      await req.response.close();
    });
    return _FakeServer._(http, hits);
  }
}

class _Harness {
  _Harness(String baseUrl) {
    client = ApiClient(
      baseUrl: baseUrl,
      readAccess: () async => access,
      readRefresh: () async => refresh,
      writeTokens: (a, r) async {
        access = a;
        refresh = r;
      },
      clearTokens: () async {
        cleared = true;
        access = null;
        refresh = null;
      },
    );
  }

  late final ApiClient client;
  String? access = 'access-1';
  String? refresh = 'refresh-1';
  bool cleared = false;
}

void main() {
  test('401 refreshes the token once, retries with it, and succeeds', () async {
    final server = await _FakeServer.start(refreshSucceeds: true);
    addTearDown(server.http.close);
    final h = _Harness(server.baseUrl);

    final me = await h.client.me();

    expect(me.id, 'user-1');
    expect(h.access, 'access-2');
    expect(h.refresh, 'refresh-2');
    expect(h.cleared, isFalse);
    expect(server.hitsForMe, 2);
    expect(server.hitsForRefresh, 1);
  });

  test(
    '401 with a failing refresh clears the session and surfaces the error',
    () async {
      final server = await _FakeServer.start(refreshSucceeds: false);
      addTearDown(server.http.close);
      final h = _Harness(server.baseUrl);

      await expectLater(
        h.client.me(),
        throwsA(isA<ApiException>().having((e) => e.status, 'status', 401)),
      );

      expect(h.cleared, isTrue);
      expect(h.access, isNull);
      expect(h.refresh, isNull);
      expect(server.hitsForMe, 1);
      expect(server.hitsForRefresh, 1);
    },
  );

  test('a second 401 on the retried request does not loop', () async {
    final server = await _FakeServer.start(
      refreshSucceeds: true,
      meAlways401: true,
    );
    addTearDown(server.http.close);
    final h = _Harness(server.baseUrl);

    await expectLater(
      h.client.me(),
      throwsA(isA<ApiException>().having((e) => e.status, 'status', 401)),
    );

    expect(server.hitsForMe, 2);
    expect(server.hitsForRefresh, 1);
  });

  test('concurrent 401s share a single refresh', () async {
    final server = await _FakeServer.start(refreshSucceeds: true);
    addTearDown(server.http.close);
    final h = _Harness(server.baseUrl);

    final profiles = await Future.wait([h.client.me(), h.client.me()]);

    expect(profiles.map((p) => p.id), ['user-1', 'user-1']);
    expect(server.hitsForRefresh, 1);
    expect(server.hitsForMe, 4); // two originals + two retries
  });
}
