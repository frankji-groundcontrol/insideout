import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/errors_l10n.dart';
import 'package:insideout/l10n/l10n.dart';

// USER-11: the mapper is the single place raw errors become UI copy —
// status → key, backoff codes → throttled, transport → network, unknown →
// generic. The fake t returns the key so the assertions pin the mapping,
// not the translations (l10n_test.dart owns key parity between en and zh).
void main() {
  String key(Object e) => localizedError(e, (k, [a]) => k);

  test('ApiException maps by status', () {
    expect(key(ApiException('bad input', status: 400)), 'error.badRequest');
    expect(key(ApiException('no bearer', status: 401)), 'error.unauthorized');
    expect(key(ApiException('nope', status: 403)), 'error.forbidden');
    expect(key(ApiException('gone', status: 404)), 'error.notFound');
    expect(key(ApiException('clash', status: 409)), 'error.conflict');
    expect(key(ApiException('boom', status: 500)), 'error.server');
    expect(key(ApiException('weird', status: 418)), 'error.generic');
  });

  test('backoff codes and 429 collapse to throttled', () {
    expect(
      key(ApiException('slow down', status: 429, code: 'APP_THROTTLE')),
      'error.throttled',
    );
    expect(
      key(ApiException('circuit', status: 503, code: 'CIRCUIT_OPEN')),
      'error.throttled',
    );
    expect(
      key(ApiException('anthropic', status: 429, code: 'ANTHROPIC_RATE_LIMIT')),
      'error.throttled',
    );
    expect(key(ApiException('plain 429', status: 429)), 'error.throttled');
  });

  test('transport failures read as network, everything else as generic', () {
    expect(key(TimeoutException('t')), 'error.network');
    expect(
      key(
        DioException(
          requestOptions: RequestOptions(),
          type: DioExceptionType.connectionTimeout,
        ),
      ),
      'error.network',
    );
    expect(key(StateError('untouched')), 'error.generic');
  });

  // validateStatus (<500) makes dio throw a badResponse DioException for every
  // 5xx, so a 502/503 on an ordinary call reached the mapper as a transport
  // failure and read "Network problem" instead of "Server hiccup".
  test('a 5xx response reads as a server fault, not a network one', () {
    DioException badResponse(int status) => DioException(
      requestOptions: RequestOptions(),
      type: DioExceptionType.badResponse,
      response: Response(requestOptions: RequestOptions(), statusCode: status),
    );

    expect(key(badResponse(503)), 'error.server');
    expect(key(badResponse(502)), 'error.server');
    // Only 5xx is a server fault: a response-less DioException is still the
    // network.
    expect(
      key(
        DioException(
          requestOptions: RequestOptions(),
          type: DioExceptionType.connectionError,
        ),
      ),
      'error.network',
    );
  });

  // rv4-FR-F03, F09, F16: the server tells a failed repo link apart by code
  // (github.go), and the status alone says the wrong thing: 400 reads "bad
  // request" and 409 reads "changed elsewhere, refresh", though refreshing
  // changes nothing and the fix for a private repo is installing the App.
  const repoCodes = <(int, String, String)>[
    (400, 'GITHUB_REPO_NOT_VISIBLE', 'error.githubRepoNotVisible'),
    (400, 'GITHUB_REPO_URL_INVALID', 'error.githubRepoUrlInvalid'),
    (409, 'GITHUB_REPO_ALREADY_LINKED', 'error.githubRepoAlreadyLinked'),
  ];

  for (final (status, code, want) in repoCodes) {
    test('$code reads as its own advice, not as a bare $status', () {
      expect(
        key(ApiException('server words', status: status, code: code)),
        want,
      );
    });

    test('$code has its own words in both languages', () {
      final en = lookup('en-US', want);
      final zh = lookup('zh-CN', want);
      expect(en, isNot(want), reason: 'the English key is missing');
      expect(zh, isNot(want), reason: 'the Chinese key is missing');
      for (final locale in ['en-US', 'zh-CN']) {
        final text = locale == 'en-US' ? en : zh;
        expect(text, isNot(lookup(locale, 'error.badRequest')));
        expect(text, isNot(lookup(locale, 'error.conflict')));
      }
    });
  }

  test('a private repo is told to install the GitHub App, then link again', () {
    expect(
      lookup('en-US', 'error.githubRepoNotVisible'),
      contains(
        'install the InsideOut GitHub App on the repository, then link it again',
      ),
    );
  });

  test('the same status without a repo-link code keeps the status copy', () {
    expect(
      key(ApiException('x', status: 400, code: 'SOMETHING_ELSE')),
      'error.badRequest',
    );
    expect(
      key(ApiException('x', status: 409, code: 'SOMETHING_ELSE')),
      'error.conflict',
    );
  });

  // The same fault off the wire: the real client against a loopback 503.
  test('a real 503 through the client reads as a server fault', () async {
    final http = await HttpServer.bind('127.0.0.1', 0);
    addTearDown(() => http.close(force: true));
    http.listen((req) async {
      req.response.statusCode = 503;
      await req.response.close();
    });
    final client = ApiClient(
      baseUrl: 'http://127.0.0.1:${http.port}/api/v1',
      readAccess: () async => null,
      readRefresh: () async => null,
      writeTokens: (a, r) async {},
      clearTokens: () async {},
    );

    Object? failure;
    try {
      await client.me();
    } catch (e) {
      failure = e;
    }

    expect(failure, isNotNull);
    expect(key(failure!), 'error.server');
  });
}
