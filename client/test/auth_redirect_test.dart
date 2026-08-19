import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/session/auth_redirect.dart';

void main() {
  test('does not redirect until hydrate has finished', () {
    expect(
      authRedirect(ready: false, signedIn: false, location: '/dashboard'),
      isNull,
    );
  });

  test('signed-out users may visit landing login and register only', () {
    expect(authRedirect(ready: true, signedIn: false, location: '/'), isNull);
    expect(authRedirect(ready: true, signedIn: false, location: '/login'), isNull);
    expect(authRedirect(ready: true, signedIn: false, location: '/register'), isNull);
    expect(authRedirect(ready: true, signedIn: false, location: '/dashboard'), '/login');
    expect(authRedirect(ready: true, signedIn: false, location: '/workspace/1'), '/login');
    expect(authRedirect(ready: true, signedIn: false, location: '/prd/1'), '/login');
  });

  test('signed-in users are sent away from login and register', () {
    expect(authRedirect(ready: true, signedIn: true, location: '/login'), '/dashboard');
    expect(authRedirect(ready: true, signedIn: true, location: '/register'), '/dashboard');
    expect(authRedirect(ready: true, signedIn: true, location: '/dashboard'), isNull);
    expect(authRedirect(ready: true, signedIn: true, location: '/'), isNull);
  });
}
