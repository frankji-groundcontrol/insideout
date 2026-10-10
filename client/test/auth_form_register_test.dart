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

// USER-16: the server answers 403 for "registration is closed", "an
// invitation is required" and "invalid or expired invitation", yet the
// register form turned every failure into "Could not create account. Please
// try again." — advice that cannot succeed when the code is the problem. A 403
// now says the invitation is the problem; any other failure keeps the
// generic copy.
class _RegisterApi extends ApiClient {
  _RegisterApi(this.failure)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final ApiException failure;

  @override
  Future<String> authConfig() async => 'invitation_only';

  @override
  Future<UserProfile> register(
    String email,
    String password,
    String username, {
    String inviteCode = '',
  }) async => throw failure;
}

typedef _Copy = ({String locale, String submit, String invite, String generic});

const _en = (
  locale: 'en-US',
  submit: 'Create account',
  invite:
      'That invitation code is missing, invalid or expired. Check it, or ask an admin for a new invitation.',
  generic: 'Could not create account. Please try again.',
);

const _zh = (
  locale: 'zh-CN',
  submit: '创建账号',
  invite: '邀请码缺失、无效或已过期。请检查邀请码，或向管理员索取新的邀请。',
  generic: '创建账号失败，请重试',
);

// Fill the form, press submit, and let the failure land.
Future<void> _submitRegister(
  WidgetTester tester,
  _Copy copy,
  ApiException failure,
) async {
  final session = Session();
  session.ready = true;
  session.api = _RegisterApi(failure);
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
  // authConfig() answers invitation_only, which reveals the code field.
  await tester.pumpAndSettle();

  // Field order: email, display name, invitation code, password.
  final fields = find.byType(TextField);
  expect(fields, findsNWidgets(4));
  await tester.enterText(fields.at(0), 'ada@example.com');
  await tester.enterText(fields.at(1), 'Ada');
  await tester.enterText(fields.at(2), 'STALE-CODE');
  await tester.enterText(fields.at(3), 'password123');
  await tester.tap(find.text(copy.submit));
  await tester.pumpAndSettle();
}

void main() {
  for (final copy in [_en, _zh]) {
    testWidgets('a rejected invitation (403) says so in ${copy.locale}', (
      tester,
    ) async {
      await _submitRegister(
        tester,
        copy,
        ApiException('invalid or expired invitation', status: 403),
      );

      expect(find.text(copy.invite), findsOneWidget);
      expect(find.text(copy.generic), findsNothing);
    });

    testWidgets('any other register failure keeps the generic copy in '
        '${copy.locale}', (tester) async {
      await _submitRegister(
        tester,
        copy,
        ApiException('internal error', status: 500),
      );

      expect(find.text(copy.generic), findsOneWidget);
      expect(find.text(copy.invite), findsNothing);
    });
  }
}
