import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/errors.dart';

void main() {
  test('maps a JSON error body to ApiException', () {
    final err = toApiError(401, {'error': 'authentication required'});
    expect(err.status, 401);
    expect(err.message, 'authentication required');
  });

  test('preserves throttle code and retry_after_seconds', () {
    final err = toApiError(429, {
      'error': 'slow down',
      'code': 'APP_THROTTLE',
      'retry_after_seconds': 12,
    });
    expect(err.code, 'APP_THROTTLE');
    expect(err.retryAfterSeconds, 12);
  });

  test('coach backoff matches Nuxt APP_THROTTLE and CIRCUIT_OPEN', () {
    final throttle = toApiError(429, {'error': 'slow', 'code': 'APP_THROTTLE', 'retry_after_seconds': 12});
    expect(isCoachBackoff(throttle), isTrue);
    expect(coachRetrySeconds(throttle), 12);
    final circuit = toApiError(503, {'error': 'open', 'code': 'CIRCUIT_OPEN'});
    expect(isCoachBackoff(circuit), isTrue);
    expect(coachRetrySeconds(circuit), 30);
    final ordinary = toApiError(400, {'error': 'nope'});
    expect(isCoachBackoff(ordinary), isFalse);
  });

  test('preserves replace_conflict liveCount from PRD build', () {
    final err = toApiError(409, {
      'error': 'roadmap has existing nodes; confirm replacement',
      'code': 'replace_conflict',
      'liveCount': 4,
    });
    expect(err.code, 'replace_conflict');
    expect(err.liveCount, 4);
  });
}
