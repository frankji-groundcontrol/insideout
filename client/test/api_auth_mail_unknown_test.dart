import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';

// PM-04: a backend that predates the mailEnabled field must read as UNKNOWN,
// not as an affirmative "mail works". The mail-off notice fires only on an
// explicit false; an absent flag neither promises nor warns, so the client
// must not fold it into true.
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

  test('a config without mailEnabled reads as unknown, not mail on', () async {
    final client = await clientFor({'signupMode': 'invitation_only'});
    expect(await client.authMailEnabled(), isNull);
  });

  test('an explicit mailEnabled still reads as given', () async {
    final off = await clientFor({'signupMode': 'open', 'mailEnabled': false});
    expect(await off.authMailEnabled(), isFalse);
    final on = await clientFor({'signupMode': 'open', 'mailEnabled': true});
    expect(await on.authMailEnabled(), isTrue);
  });
}
