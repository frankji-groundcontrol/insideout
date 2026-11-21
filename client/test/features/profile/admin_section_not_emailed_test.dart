import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/features/profile/admin_section.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv4-USER-07: the invitation dialog told the admin the code "was emailed to
// the invitee" no matter what happened to the mail. The server now reports
// mailSent; when it is false (mail disabled, or the provider rejected the
// message) the dialog must say the invitee was NOT emailed and point at the
// signup link as the thing to send.
class _InviteApi extends ApiClient {
  _InviteApi(this.created)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final Map<String, dynamic> created;

  @override
  Future<String> authConfig() async => 'invitation_only';

  @override
  Future<List<Map<String, dynamic>>> listInvitations() async => const [];

  @override
  Future<Map<String, dynamic>> createInvitation(String email) async => created;
}

Future<void> _createInvitationDialog(
  WidgetTester tester,
  Map<String, dynamic> created,
) async {
  final session = Session();
  session.ready = true;
  session.api = _InviteApi(created);
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
      child: const MaterialApp(
        home: Scaffold(body: SingleChildScrollView(child: AdminSection())),
      ),
    ),
  );
  await tester.pumpAndSettle();

  await tester.enterText(find.byType(TextField), 'ada@example.com');
  await tester.tap(find.text('Invite'));
  await tester.pumpAndSettle();
}

void main() {
  const link = 'https://app.example.com/register?invite=one-time-code';

  testWidgets('mailSent false tells the admin to send the link themselves', (
    tester,
  ) async {
    await _createInvitationDialog(tester, {
      'code': 'one-time-code',
      'signupUrl': link,
      'mailSent': false,
    });

    expect(find.textContaining('was not sent'), findsOneWidget);
    expect(find.textContaining('was emailed'), findsNothing);
    // The link stays visible and copyable: it is now the only delivery path.
    expect(find.text(link), findsOneWidget);
    expect(find.text('Copy signup link'), findsOneWidget);
  });

  testWidgets('mailSent true keeps saying the invitee was emailed', (
    tester,
  ) async {
    await _createInvitationDialog(tester, {
      'code': 'one-time-code',
      'signupUrl': link,
      'mailSent': true,
    });

    expect(find.textContaining('was emailed to the invitee'), findsOneWidget);
    expect(find.textContaining('was not sent'), findsNothing);
  });

  testWidgets('a server that predates mailSent makes no not-emailed claim', (
    tester,
  ) async {
    // Same convention as authMailEnabled: only an explicit false warns.
    await _createInvitationDialog(tester, {
      'code': 'one-time-code',
      'signupUrl': link,
    });

    expect(find.textContaining('was emailed to the invitee'), findsOneWidget);
    expect(find.textContaining('was not sent'), findsNothing);
  });
}
