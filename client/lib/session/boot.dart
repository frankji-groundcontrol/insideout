import 'package:flutter/material.dart';

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

/// The boot loading page: 印 stamp over an ink ring spinner. Deliberately
/// text-free — appearance (locale) may not be hydrated yet, and on web the
/// CJK fonts the gate exists for are not loaded yet.
class BootLoadingPage extends StatelessWidget {
  const BootLoadingPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Theme.of(context).scaffoldBackgroundColor,
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Image.asset(
              'assets/seals/yin.webp',
              width: 72,
              height: 72,
              filterQuality: FilterQuality.medium,
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
