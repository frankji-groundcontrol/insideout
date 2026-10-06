import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/secret_query.dart';

void main() {
  test('url rewrite drops invite, reset, and magic-link secrets', () {
    expect(
      urlWithoutQueryKeys(
        'https://insideout.example/register?invite=sekret',
        const ['invite'],
      ),
      'https://insideout.example/register',
    );
    expect(
      urlWithoutQueryKeys(
        'https://insideout.example/reset-password?token=sekret&lang=zh',
        const ['token'],
      ),
      'https://insideout.example/reset-password?lang=zh',
    );
    expect(
      urlWithoutQueryKeys('/magic-link?token=one-time#done', const ['token']),
      '/magic-link#done',
    );
    expect(
      urlWithoutQueryKeys('/reset-password?lang=zh&token=sekret&next=1', const [
        'token',
      ]),
      '/reset-password?lang=zh&next=1',
    );
    expect(
      urlWithoutQueryKeys('/reset-password?token=a%2Fb', const ['token']),
      '/reset-password',
    );
  });

  test('url rewrite leaves unrelated queries untouched', () {
    const login = '/login?next=/dashboard';
    expect(urlWithoutQueryKeys(login, const ['token', 'invite']), login);
    const inviteCode = '/register?inviteCode=abc';
    expect(urlWithoutQueryKeys(inviteCode, const ['invite']), inviteCode);
    expect(
      urlWithoutQueryKeys('/register?invite=sekret', const []),
      '/register?invite=sekret',
    );
  });

  test('stripped urls do not still contain the secret', () {
    for (final url in [
      urlWithoutQueryKeys('/register?invite=sekret', const ['invite']),
      urlWithoutQueryKeys('/reset-password?token=sekret', const ['token']),
      urlWithoutQueryKeys('/magic-link?token=sekret', const ['token']),
    ]) {
      expect(url.contains('sekret'), isFalse);
      expect(url.contains('?'), isFalse);
    }
  });
}
