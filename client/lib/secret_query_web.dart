import 'package:web/web.dart' as web;

import 'secret_query.dart';

/// Replaces the current history entry so [keys] leave the address bar.
///
/// Keeps the existing history state so the web engine's back-button serial
/// count stays intact. Does not use GoRouter.
void scrubBrowserQueryKeys(Iterable<String> keys) {
  final location = web.window.location;
  final history = web.window.history;
  final next = urlWithoutQueryKeys(location.href, keys);
  if (next == location.href) return;
  history.replaceState(history.state, '', next);
}
