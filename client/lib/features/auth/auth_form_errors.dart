import '../../api/errors.dart';

/// The l10n key that says why a register attempt failed.
///
/// The server answers 403 both when registration is closed and when the
/// invitation is missing, wrong or expired, and the body carries no code to
/// tell them apart. The sign-up mode the form fetched from /auth/config is the
/// structured signal: a closed server gets the closed copy, and every other
/// 403, including one that arrives before the mode did, keeps the invitation
/// copy. Anything that is not a 403 stays the generic "try again".
String registerErrorKey(Object error, String? signupMode) {
  if (error is ApiException && error.status == 403) {
    return signupMode == 'closed'
        ? 'register.closedError'
        : 'register.inviteError';
  }
  return 'register.error';
}
