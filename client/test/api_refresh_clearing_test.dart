import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/sse.dart';

/// FULL-05: a failed refresh clears the session only when the server says the
/// session is over (401/403), the rule hydrate() already follows. A dropped
/// connection or a 5xx says nothing about the refresh token, and signing the
/// user out for it throws away a working login.

typedef _Handler = Future<void> Function(HttpRequest req);

Future<void> _reply(HttpRequest req, int status) async {
  req.response
    ..statusCode = status
    ..headers.contentType = ContentType.json
    ..write(jsonEncode({'error': 'status $status'}));
  await req.response.close();
}

Future<void> _drop(HttpRequest req) async {
  (await req.response.detachSocket()).destroy();
}

/// Every path answers 401 (the access token is dead); /auth/refresh is [refresh].
Future<HttpServer> _serve(_Handler refresh) async {
  final server = await HttpServer.bind('127.0.0.1', 0);
  server.listen((req) async {
    await utf8.decoder.bind(req).join();
    if (req.uri.path == '/api/v1/auth/refresh') {
      await refresh(req);
    } else {
      await _reply(req, 401);
    }
  });
  addTearDown(server.close);
  return server;
}

({ApiClient client, bool Function() cleared}) _clientFor(
  HttpServer server, {
  String? refreshToken = 'refresh-1',
}) {
  var cleared = false;
  final client = ApiClient(
    baseUrl: 'http://127.0.0.1:${server.port}/api/v1',
    readAccess: () async => 'access-1',
    readRefresh: () async => refreshToken,
    writeTokens: (a, r) async {},
    clearTokens: () async => cleared = true,
  );
  return (client: client, cleared: () => cleared);
}

Matcher _status401() =>
    throwsA(isA<ApiException>().having((e) => e.status, 'status', 401));

void main() {
  group('interceptor refresh', () {
    test('a 503 from /auth/refresh keeps the session', () async {
      final h = _clientFor(await _serve((req) => _reply(req, 503)));

      await expectLater(h.client.me(), _status401());

      expect(h.cleared(), isFalse);
    });

    test('a dropped connection during refresh keeps the session', () async {
      final h = _clientFor(await _serve(_drop));

      await expectLater(h.client.me(), _status401());

      expect(h.cleared(), isFalse);
    });

    for (final status in [401, 403]) {
      test('a $status from /auth/refresh clears the session', () async {
        final h = _clientFor(await _serve((req) => _reply(req, status)));

        await expectLater(h.client.me(), _status401());

        expect(h.cleared(), isTrue);
      });
    }

    test('having no refresh token clears the session', () async {
      final h = _clientFor(
        await _serve((req) => _reply(req, 200)),
        refreshToken: null,
      );

      await expectLater(h.client.me(), _status401());

      expect(h.cleared(), isTrue);
    });
  });

  group('coachSseWithRefresh', () {
    Future<SseAttempt> open() async =>
        const SseAttempt(status: 401, chunks: Stream.empty());

    for (final (name, boom) in <(String, Object Function())>[
      ('a network error', () => Exception('socket closed')),
      ('a 503', () => ApiException('unavailable', status: 503)),
    ]) {
      test('$name during refresh keeps the session', () async {
        var cleared = false;

        await expectLater(
          coachSseWithRefresh(
            open,
            () async => throw boom(),
            () async => cleared = true,
          ).drain<void>(),
          throwsA(isA<ApiException>()),
        );

        expect(cleared, isFalse);
      });
    }

    for (final status in [401, 403]) {
      test('a $status during refresh clears the session', () async {
        var cleared = false;

        await expectLater(
          coachSseWithRefresh(
            open,
            () async => throw ApiException('rejected', status: status),
            () async => cleared = true,
          ).drain<void>(),
          throwsA(isA<ApiException>()),
        );

        expect(cleared, isTrue);
      });
    }

    // rv4-FR-F01: the roadmap page's retry loop ends for good on 401, 403 and
    // 404 (a signed-out user, a gone project). A kept session must throw
    // something it retries, or live presence and progress stop for a user who
    // is still signed in. 401 stays for the paths that cleared the session.
    Matcher retryable() => throwsA(
      isA<ApiException>().having(
        (e) => e.status,
        'status',
        isNot(anyOf(401, 403, 404)),
      ),
    );

    for (final (name, boom) in <(String, Object Function())>[
      ('a network error', () => Exception('socket closed')),
      ('a 503', () => ApiException('unavailable', status: 503)),
      ('an answer with no status', () => ApiException('no tokens in reply')),
    ]) {
      test(
        '$name during refresh throws a status a retry loop retries',
        () async {
          await expectLater(
            coachSseWithRefresh(
              open,
              () async => throw boom(),
              () async {},
            ).drain<void>(),
            retryable(),
          );
        },
      );
    }

    test('a refresh refused for rate (429) keeps its own status', () async {
      await expectLater(
        coachSseWithRefresh(
          open,
          () async => throw ApiException('slow down', status: 429),
          () async {},
        ).drain<void>(),
        throwsA(isA<ApiException>().having((e) => e.status, 'status', 429)),
      );
    });
  });
}
