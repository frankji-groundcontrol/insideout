import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/features/auth/auth_door.dart';
import 'package:insideout/features/auth/auth_form.dart';
import 'package:insideout/features/auth/forgot_password_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// USER-04: with no INSIDEOUT_MAIL_API_KEY the server only logs magic links,
// sign-in codes and reset mails, yet the sign-in UI offered all three without a
// word. The server reports mailEnabled in its public config; when it is false
// the login form and the forgot-password page say so. The options stay —
// annotated, never hidden — and a failed config call assumes mail works.
class _MailApi extends ApiClient {
  _MailApi(this.mail)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// true: mail is delivered; false: dev mode; null: the config call fails.
  final bool? mail;

  @override
  Future<bool> authMailEnabled() async {
    final delivered = mail;
    if (delivered == null) throw ApiException('boom', status: 500);
    return delivered;
  }
}

typedef _Copy = ({
  String locale,
  String notice,
  String magic,
  String code,
  String forgot,
});

const _en = (
  locale: 'en-US',
  notice:
      'Email is not configured on this server: sign-in links, codes and password-reset emails are only written to the server log.',
  magic: 'Email link',
  code: 'Email code',
  forgot: 'Forgot password?',
);

const _zh = (
  locale: 'zh-CN',
  notice: '此服务器未配置邮件发送：登录链接、验证码和重置密码邮件只会写入服务器日志，不会真正发出。',
  magic: '邮件链接',
  code: '验证码',
  forgot: '忘记密码？',
);

Future<void> _pump(
  WidgetTester tester,
  _Copy copy,
  bool? mail,
  Widget home,
) async {
  final session = Session();
  session.ready = true;
  session.api = _MailApi(mail);
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
      child: MaterialApp(home: home),
    ),
  );
  await tester.pumpAndSettle();
}

Widget _login() => AuthDoor(
  greeting: 'Log in',
  onClose: () {},
  child: const AuthForm(prompt: AuthPrompt.login),
);

void main() {
  for (final copy in [_en, _zh]) {
    group('mail in dev mode, ${copy.locale}', () {
      testWidgets('the login form annotates the emailed options', (
        tester,
      ) async {
        await _pump(tester, copy, false, _login());

        expect(find.text(copy.notice), findsOneWidget);
        // Annotated, not hidden.
        expect(find.text(copy.magic), findsOneWidget);
        expect(find.text(copy.code), findsOneWidget);
        expect(find.text(copy.forgot), findsOneWidget);
      });

      testWidgets('the forgot-password page says the reset mail is not sent', (
        tester,
      ) async {
        await _pump(tester, copy, false, const ForgotPasswordPage());

        expect(find.text(copy.notice), findsOneWidget);
      });
    });

    group('mail delivered or unknown, ${copy.locale}', () {
      for (final (name, mail) in [
        ('configured', true),
        ('config call fails', null),
      ]) {
        testWidgets('login shows no notice when $name', (tester) async {
          await _pump(tester, copy, mail, _login());

          expect(find.text(copy.notice), findsNothing);
          expect(find.text(copy.magic), findsOneWidget);
          expect(find.text(copy.code), findsOneWidget);
          expect(find.text(copy.forgot), findsOneWidget);
        });

        testWidgets('forgot-password shows no notice when $name', (
          tester,
        ) async {
          await _pump(tester, copy, mail, const ForgotPasswordPage());

          expect(find.text(copy.notice), findsNothing);
        });
      }
    });
  }
}
