import 'dart:async';

import 'package:flutter/widgets.dart';

import 'secret_query_stub.dart'
    if (dart.library.ui_web) 'secret_query_web.dart';

/// Drops [keys] from the browser history entry.
///
/// GoRouter.replace would rebuild the page with an empty secret, so web uses
/// history.replaceState only. The router writes the query back in a post-frame
/// platform message; the rewrite runs on the microtask after that write.
void scrubSecretQueryKeys(Iterable<String> keys) {
  WidgetsBinding.instance.addPostFrameCallback((_) {
    scheduleMicrotask(() => scrubBrowserQueryKeys(keys));
  });
}
