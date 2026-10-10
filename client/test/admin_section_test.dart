import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/features/profile/admin_section.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// FS-08: the revoke confirm is its own route, so it can outlive the section
// when the user leaves the page with it open. Confirming it afterwards must
// not call setState on the disposed state.
class _AdminApi extends ApiClient {
  _AdminApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<String> authConfig() async => 'invitation_only';

  @override
  Future<List<Map<String, dynamic>>> listInvitations() async => [
    {
      'id': 'inv1',
      'email': 'ada@example.com',
      'usedAt': null,
      'revokedAt': null,
    },
  ];
}

void main() {
  testWidgets('confirming a revoke after the section is gone does not throw', (
    tester,
  ) async {
    final session = Session();
    session.ready = true;
    session.api = _AdminApi();
    final shown = ValueNotifier(true);
    addTearDown(shown.dispose);
    await tester.binding.setSurfaceSize(const Size(900, 1200));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(
            create: (_) =>
                Appearance(store: MemorySettingsStore(), locale: 'en-US'),
          ),
        ],
        child: MaterialApp(
          home: Scaffold(
            body: ValueListenableBuilder<bool>(
              valueListenable: shown,
              builder: (_, on, _) => on
                  ? const SingleChildScrollView(child: AdminSection())
                  : const SizedBox(),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Revoke'));
    await tester.pumpAndSettle();
    // The user leaves the page with the confirm dialog still open.
    shown.value = false;
    await tester.pump();
    await tester.tap(find.text('Revoke').last);
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
  });
}
