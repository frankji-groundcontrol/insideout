class ApiException implements Exception {
  ApiException(
    this.message, {
    this.status,
    this.code,
    this.retryAfterSeconds,
    this.liveCount,
    this.evidenceCount,
    this.doneCount,
    this.validatedCount,
  });

  final String message;
  final int? status;
  final String? code;
  final int? retryAfterSeconds;
  final int? liveCount;

  /// What a roadmap replace would erase (409 replace_conflict and
  /// roadmap_replace_has_history).
  final int? evidenceCount;
  final int? doneCount;
  final int? validatedCount;

  @override
  String toString() => message;
}

ApiException toApiError(int? status, Map<String, dynamic> body) {
  final code = body['code'] as String?;
  final message = (body['error'] as String?) ?? 'Request failed';
  final retry = body['retryAfterSeconds'];
  int? retryAfter;
  if (retry is num) retryAfter = retry.toInt();
  int? count(String key) {
    final v = body[key];
    return v is num ? v.toInt() : null;
  }

  return ApiException(
    message,
    status: status,
    code: code,
    retryAfterSeconds: retryAfter,
    liveCount: count('liveCount'),
    evidenceCount: count('evidenceCount'),
    doneCount: count('doneCount'),
    validatedCount: count('validatedCount'),
  );
}

/// Codes that put a Coach send on backoff (countdown, composer locked).
bool isCoachBackoff(ApiException e) {
  return e.code == 'app_throttle' ||
      e.code == 'circuit_open' ||
      e.code == 'anthropic_rate_limit';
}

/// A Coach backoff because the AI service is down or saturated (circuit open,
/// queue busy), not because the caller was rate limited. Read from the code,
/// never from the English message.
bool isCoachUnavailable(ApiException e) => e.code == 'circuit_open';

int coachRetrySeconds(ApiException e) {
  if (e.retryAfterSeconds != null) return e.retryAfterSeconds!;
  return e.code == 'app_throttle' ? 60 : 30;
}
