import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/errors_l10n.dart';

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
