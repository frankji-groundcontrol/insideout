import 'dart:async';

import 'package:dio/dio.dart';

import 'errors.dart';

/// USER-11 (review 2): the UI is zh-CN-first, but every FutureBuilder
/// error branch rendered the raw English server text as page content.
/// One mapper, routed through all data-page error branches: ApiException
/// maps by status (with the coach-backoff codes collapsed to one string),
/// transport failures map to "network", everything else to a generic
/// apology. Auth forms deliberately do NOT use this — a login 401 means
/// "wrong credentials", not "session expired", and deserves its own copy.
typedef TFn = String Function(String key, [Map<String, Object?>? args]);

String localizedError(Object error, TFn t) {
  if (error is ApiException) {
    if (isCoachBackoff(error) || error.status == 429) {
      return t('error.throttled');
    }
    switch (error.status) {
      case 400:
        return t('error.badRequest');
      case 401:
        return t('error.unauthorized');
      case 403:
        return t('error.forbidden');
      case 404:
        return t('error.notFound');
      case 409:
        return t('error.conflict');
    }
    if (error.status != null && error.status! >= 500) {
      return t('error.server');
    }
    return t('error.generic');
  }
  if (error is DioException || error is TimeoutException) {
    return t('error.network');
  }
  return t('error.generic');
}
