import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/session/token_store.dart';

// rv4-FR-F212, F209, F11: a release build must not send tokens in clear text.
// rv4-COMP-06 set android:usesCleartextTraffic="false", but that flag and a
// network security config govern the platform HTTP stack only. The client's dio
// runs on dart:io sockets, which never consult them, so a release APK built
// with --dart-define=API_BASE=http://<lan host>/... still sent the bearer and
// refresh tokens in the clear. The guard is Dart: checkApiBase, run by Session.
//
// kReleaseMode is a compile-time constant and `flutter test` is always a debug
// build, so the checks take the mode as an argument.
const _lan = 'http://192.168.1.5:8080/api/v1';

void main() {
  group('checkApiBase in a release build', () {
    test('refuses http to a host that is not this device', () {
      for (final base in [
        _lan,
        'http://api.example.org/api/v1',
        'http://10.0.2.20:8080/api/v1',
        'http://0.0.0.0:8080/api/v1',
        // Look-alikes: the host is what follows the @, or only starts like one.
        'http://localhost@evil.example/api/v1',
        'http://127.0.0.1.example.org/api/v1',
      ]) {
        expect(
          () => checkApiBase(base, releaseMode: true),
          throwsA(
            isA<StateError>().having(
              (e) => e.message,
              'message',
              allOf(contains('API_BASE'), contains('https')),
            ),
          ),
          reason: base,
        );
      }
    });

    test('refuses what it cannot prove is https or a plain path', () {
      for (final base in [
        'ftp://files.example.org/api',
        '//cdn.example.org/api/v1', // takes the page's scheme, unknowable here
        'localhost:8080/api/v1', // parses with the scheme "localhost"
        'http://[::1', // not a URL
      ]) {
        expect(
          () => checkApiBase(base, releaseMode: true),
          throwsStateError,
          reason: base,
        );
      }
    });

    test('keeps https, a path, and http to the device itself', () {
      for (final base in [
        'https://api.example.org/api/v1',
        'HTTPS://API.EXAMPLE.ORG/api/v1',
        '/api/v1', // the hosted web build: same origin behind nginx
        'http://127.0.0.1:8080/api/v1',
        'http://localhost:8080/api/v1',
        'http://[::1]:8080/api/v1',
        'http://10.0.2.2:8080/api/v1', // the Android emulator's alias for the host
      ]) {
        expect(
          () => checkApiBase(base, releaseMode: true),
          returnsNormally,
          reason: base,
        );
      }
    });
  });

  test('debug and profile builds keep every base', () {
    // kReleaseMode is false in both, so releaseMode: false stands for either.
    for (final base in [_lan, 'http://api.example.org/api/v1', 'ftp://x/y']) {
      expect(
        () => checkApiBase(base, releaseMode: false),
        returnsNormally,
        reason: base,
      );
    }
  });

  group('Session', () {
    test('refuses an http LAN base in a release build', () {
      expect(
        () => Session(
          store: MemoryTokenStore(),
          apiBase: _lan,
          releaseMode: true,
        ),
        throwsA(
          isA<StateError>().having(
            (e) => e.message,
            'message',
            allOf(contains('API_BASE'), contains('https')),
          ),
        ),
      );
    });

    test('builds its client against an https base in a release build', () {
      final session = Session(
        store: MemoryTokenStore(),
        apiBase: 'https://api.example.org/api/v1',
        releaseMode: true,
      );
      expect(session.api.baseUrl, 'https://api.example.org/api/v1');
    });

    test('keeps the http LAN base in a debug build', () {
      final session = Session(store: MemoryTokenStore(), apiBase: _lan);
      expect(session.api.baseUrl, _lan);
    });

    test('the default base survives a release build', () {
      expect(
        () => Session(store: MemoryTokenStore(), releaseMode: true),
        returnsNormally,
      );
    });
  });

  test('the manifest does not promise what dart:io ignores', () {
    final manifest = File(
      'android/app/src/main/AndroidManifest.xml',
    ).readAsStringSync();
    final comment = RegExp(r'<!--(.*?)-->', dotAll: true)
        .allMatches(manifest)
        .map((m) => m.group(1)!)
        .firstWhere((c) => c.contains('rv4-COMP-06'));
    expect(comment, contains('dart:io'));
    expect(comment, contains('checkApiBase'));
    expect(
      File('lib/session/session.dart').readAsStringSync(),
      contains('void checkApiBase('),
      reason: 'the manifest comment names the guard that does the work',
    );
  });
}
