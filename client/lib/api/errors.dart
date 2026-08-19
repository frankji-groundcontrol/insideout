class ApiException implements Exception {
  ApiException(this.message, {this.status, this.code, this.retryAfterSeconds, this.liveCount});

  final String message;
  final int? status;
  final String? code;
  final int? retryAfterSeconds;
  final int? liveCount;

  @override
  String toString() => message;
}

ApiException toApiError(int? status, Map<String, dynamic> body) {
  final code = body['code'] as String?;
  final message = (body['error'] as String?) ?? 'Request failed';
  final retry = body['retry_after_seconds'];
  int? retryAfter;
  if (retry is num) retryAfter = retry.toInt();
  final live = body['liveCount'];
  int? liveCount;
  if (live is num) liveCount = live.toInt();
  return ApiException(
    message,
    status: status,
    code: code,
    retryAfterSeconds: retryAfter,
    liveCount: liveCount,
  );
}

/// Codes Nuxt treats as a Coach send backoff (countdown, composer locked).
bool isCoachBackoff(ApiException e) {
  return e.code == 'APP_THROTTLE' || e.code == 'CIRCUIT_OPEN' || e.code == 'ANTHROPIC_RATE_LIMIT';
}

int coachRetrySeconds(ApiException e) {
  if (e.retryAfterSeconds != null) return e.retryAfterSeconds!;
  return e.code == 'APP_THROTTLE' ? 60 : 30;
}
