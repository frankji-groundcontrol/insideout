import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
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
}
