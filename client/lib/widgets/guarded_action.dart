import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../api/errors.dart';
import '../api/errors_l10n.dart';
import '../session/appearance.dart';

/// Register 4.1: write actions used to fail silently — a bare `await api.…`
/// in a button handler let the exception skip the reload and show nothing.
/// Run [write] inside this guard instead: on failure it shows a localized
/// error SnackBar and returns false so callers keep their input; on success
/// it returns true so callers reload. Modeled on the updates feed's
/// `_decide` (project_updates_feed.dart).
///
/// AI writes (roadmap build/expand) answered with a backoff code get the
/// coach's own copy — service-down vs rate-limited, with the countdown —
/// instead of a generic save-failed line (the throttled-roadmap-copy issue).
Future<bool> guardWrite(
  BuildContext context,
  Future<void> Function() write,
) async {
  try {
    await write();
  } on ApiException catch (e) {
    if (context.mounted) {
      final t = context.read<Appearance>().t;
      if (isCoachBackoff(e)) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              t(
                isCoachUnavailable(e)
                    ? 'coach.serviceUnavailable'
                    : 'coach.rateLimited',
                {'seconds': coachRetrySeconds(e)},
              ),
            ),
          ),
        );
        return false;
      }
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(t('write.failed', {'error': localizedError(e, t)})),
        ),
      );
    }
    return false;
  } on Object catch (e) {
    if (context.mounted) {
      final t = context.read<Appearance>().t;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(t('write.failed', {'error': localizedError(e, t)})),
        ),
      );
    }
    return false;
  }
  return true;
}
