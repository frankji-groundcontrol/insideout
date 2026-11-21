import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/errors.dart';

void main() {
  test('maps a JSON error body to ApiException', () {
    final err = toApiError(401, {'error': 'authentication required'});
    expect(err.status, 401);
    expect(err.message, 'authentication required');
  });

  test('preserves throttle code and retryAfterSeconds', () {
    final err = toApiError(429, {
      'error': 'slow down',
      'code': 'app_throttle',
      'retryAfterSeconds': 12,
    });
    expect(err.code, 'app_throttle');
    expect(err.retryAfterSeconds, 12);
  });

  test('coach backoff matches the app_throttle and circuit_open contract', () {
    final throttle = toApiError(429, {
      'error': 'slow',
      'code': 'app_throttle',
      'retryAfterSeconds': 12,
    });
    expect(isCoachBackoff(throttle), isTrue);
    expect(coachRetrySeconds(throttle), 12);
    final circuit = toApiError(503, {'error': 'open', 'code': 'circuit_open'});
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

  test('preserves the roadmap-history counts from a build confirm', () {
    final err = toApiError(409, {
      'error': 'history',
      'code': 'roadmap_replace_has_history',
      'liveCount': 3,
      'evidenceCount': 2,
      'doneCount': 1,
      'validatedCount': 4,
    });
    expect(err.code, 'roadmap_replace_has_history');
    expect(
      [err.liveCount, err.evidenceCount, err.doneCount, err.validatedCount],
      [3, 2, 1, 4],
    );
  });
}
