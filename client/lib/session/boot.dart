import 'package:flutter/material.dart';

import '../theme/cobalt.dart';

/// Whole-app boot gate — the reimbursement pattern: the app renders
/// immediately, but nothing product-ish mounts until session, appearance,
/// and fonts settle; until then [BootLoadingPage] runs. Completing on
/// error as well as success: a failed hydrate must lift the gate, not
/// spin forever.
class BootController extends ChangeNotifier {
  bool _done = false;

  bool get done => _done;

  void complete() {
    if (_done) return;
    _done = true;
    notifyListeners();
  }
}

/// Text-free boot mark: a cobalt disc + spinner. Appearance (locale) may
/// not be hydrated yet, and on web the CJK fonts the gate exists for are
/// not loaded yet.
class BootLoadingPage extends StatelessWidget {
  const BootLoadingPage({super.key});

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Scaffold(
      backgroundColor: palette.surfaceBase,
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              key: const Key('boot-mark'),
              width: 64,
              height: 64,
              decoration: const BoxDecoration(color: Cobalt.hue500, shape: BoxShape.circle),
            ),
            const SizedBox(height: 28),
            const SizedBox(
              width: 34,
              height: 34,
              child: CircularProgressIndicator(
                key: Key('boot-spinner'),
                strokeWidth: 3,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
