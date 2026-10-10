import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';

// USER-04: the client reads mailEnabled from the public auth config next to
// signupMode. A server that omits the flag must read as "mail works" (the
// notice is never shown on a guess), and the signupMode reader the admin
// section relies on must not care about the extra key.
void main() {
  Future<ApiClient> clientFor(Map<String, Object?> config) async {
    final http = await HttpServer.bind('127.0.0.1', 0);
    addTearDown(() => http.close(force: true));
    http.listen((req) async {
      req.response
        ..statusCode = 200
        ..headers.contentType = ContentType.json
        ..write(jsonEncode(config));
      await req.response.close();
    });
    return ApiClient(
      baseUrl: 'http://127.0.0.1:${http.port}/api/v1',
      readAccess: () async => null,
      readRefresh: () async => null,
      writeTokens: (a, r) async {},
      clearTokens: () async {},
    );
  }

  test('mailEnabled false reads as mail off, true as mail on', () async {
    final off = await clientFor({'signupMode': 'open', 'mailEnabled': false});
    expect(await off.authMailEnabled(), isFalse);
    final on = await clientFor({'signupMode': 'open', 'mailEnabled': true});
    expect(await on.authMailEnabled(), isTrue);
  });

  test('a config without the flag reads as mail on', () async {
    final client = await clientFor({'signupMode': 'invitation_only'});
    expect(await client.authMailEnabled(), isTrue);
  });

  test('authConfig still reads signupMode beside the mail flag', () async {
    final client = await clientFor({
      'signupMode': 'closed',
      'mailEnabled': false,
    });
    expect(await client.authConfig(), 'closed');
  });
}
