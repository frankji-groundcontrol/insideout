import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/auth/auth_door.dart';
import 'package:insideout/features/auth/auth_form.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv4-USER-12: the register endpoint answers 403 both when registration is
// closed and when the invitation is missing, wrong or expired, and the body
// carries no code to tell them apart. Every 403 showed the invitation copy,
// so a visitor to a closed server was told to check a code that cannot help.
// The form already fetches the sign-up mode from /auth/config; that is the
// structured signal, and the copy follows it.
class _ModeApi extends ApiClient {
  _ModeApi(this.mode, this.failure)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// The sign-up mode /auth/config answers; null makes the fetch fail.
  final String? mode;
  final ApiException failure;

  @override
  Future<String> authConfig() async =>
      mode ?? (throw ApiException('config unavailable', status: 503));

  @override
  Future<UserProfile> register(
    String email,
    String password,
    String username, {
    String inviteCode = '',
  }) async => throw failure;
}

typedef _Copy = ({
  String locale,
  String submit,
  String closed,
  String invite,
  String generic,
});

const _en = (
  locale: 'en-US',
  submit: 'Create account',
  closed:
      "Registration is closed on this server. Ask the server's administrator to open sign-up.",
  invite:
      'That invitation code is missing, invalid or expired. Check it, or ask an admin for a new invitation.',
  generic: 'Could not create account. Please try again.',
);

const _zh = (
  locale: 'zh-CN',
  submit: '创建账号',
  closed: '此服务器已关闭注册。请联系服务器管理员开放注册。',
  invite: '邀请码缺失、无效或已过期。请检查邀请码，或向管理员索取新的邀请。',
  generic: '创建账号失败，请重试',
);

// Fill the form, press submit and let the failure land. The invitation field
// shows or hides with the mode, so the fields are picked by position: email
// first, display name second, password last.
Future<void> _submit(
  WidgetTester tester,
  _Copy copy,
  String? mode,
  ApiException failure,
) async {
  final session = Session();
  session.ready = true;
  session.api = _ModeApi(mode, failure);
  await tester.binding.setSurfaceSize(const Size(900, 1200));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: copy.locale),
        ),
      ],
      child: MaterialApp(
        home: AuthDoor(
          greeting: 'Register',
          onClose: () {},
          child: const AuthForm(prompt: AuthPrompt.register),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle(); // /auth/config answers (or fails)

  final fields = find.byType(TextField);
  await tester.enterText(fields.first, 'ada@example.com');
  await tester.enterText(fields.at(1), 'Ada');
  await tester.enterText(fields.last, 'password123');
  await tester.tap(find.text(copy.submit));
  await tester.pumpAndSettle();
}

void main() {
  for (final copy in [_en, _zh]) {
    final forbidden = ApiException('registration is closed', status: 403);

    testWidgets('a 403 on a closed server says registration is closed in '
        '${copy.locale}', (tester) async {
      await _submit(tester, copy, 'closed', forbidden);

      expect(find.text(copy.closed), findsOneWidget);
      expect(find.text(copy.invite), findsNothing);
      expect(find.text(copy.generic), findsNothing);
    });

    testWidgets('a 403 on an invitation-only server blames the invitation in '
        '${copy.locale}', (tester) async {
      await _submit(tester, copy, 'invitation_only', forbidden);

      expect(find.text(copy.invite), findsOneWidget);
      expect(find.text(copy.closed), findsNothing);
    });

    testWidgets('a 403 while the mode is unknown never claims the door is '
        'closed in ${copy.locale}', (tester) async {
      await _submit(tester, copy, null, forbidden);

      expect(find.text(copy.invite), findsOneWidget);
      expect(find.text(copy.closed), findsNothing);
    });

    testWidgets('a server error on a closed server keeps the generic copy in '
        '${copy.locale}', (tester) async {
      await _submit(
        tester,
        copy,
        'closed',
        ApiException('internal error', status: 500),
      );

      expect(find.text(copy.generic), findsOneWidget);
      expect(find.text(copy.closed), findsNothing);
    });
  }
}
