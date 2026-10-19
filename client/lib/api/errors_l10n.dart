import 'dart:async';

import 'package:dio/dio.dart';

import 'errors.dart';

/// USER-11 (review 2): the UI is zh-CN-first, but every FutureBuilder
/// error branch rendered the raw English server text as page content.
/// One mapper, routed through all data-page error branches: ApiException
/// maps by status (with the coach-backoff codes collapsed to one string and
/// the repo-link codes mapped by code), transport failures map to "network",
/// everything else to a generic apology. Auth forms deliberately do NOT use
/// this — a login 401 means "wrong credentials", not "session expired", and
/// deserves its own copy.
typedef TFn = String Function(String key, [Map<String, Object?>? args]);

/// Codes whose advice the status cannot give (rv4-FR-F03, F09, F16): a repo
/// GitHub does not show the server needs the App installed, a repo another
/// project holds is not fixed by a refresh, and a stored link that is no
/// GitHub address must be linked again. Checked before the status.
const _codeKeys = <String, String>{
  'GITHUB_REPO_NOT_VISIBLE': 'error.githubRepoNotVisible',
  'GITHUB_REPO_URL_INVALID': 'error.githubRepoUrlInvalid',
  'GITHUB_REPO_ALREADY_LINKED': 'error.githubRepoAlreadyLinked',
};

String localizedError(Object error, TFn t) {
  if (error is ApiException) {
    if (isCoachBackoff(error) || error.status == 429) {
      return t('error.throttled');
    }
    final byCode = _codeKeys[error.code];
    if (byCode != null) return t(byCode);
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
  // validateStatus (<500) makes dio throw for every 5xx, so a server fault
  // reaches here as a DioException; it must not read as a dead connection.
  if (error is DioException && (error.response?.statusCode ?? 0) >= 500) {
    return t('error.server');
  }
  if (error is DioException || error is TimeoutException) {
    return t('error.network');
  }
  return t('error.generic');
}
