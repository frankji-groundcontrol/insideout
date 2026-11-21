import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';

// Plain tests only: a file that also holds a testWidgets gets a mocked
// HttpClient (every request answers 400), and this one wants real loopback.
void main() {
  test('a cursor post names the client it speaks for', () async {
    final seen = <Uri>[];
    final http = await HttpServer.bind('127.0.0.1', 0);
    addTearDown(() => http.close(force: true));
    http.listen((req) async {
      await utf8.decoder.bind(req).join();
      seen.add(req.uri);
      req.response.statusCode = 204;
      await req.response.close();
    });

    await ApiClient(
      baseUrl: 'http://127.0.0.1:${http.port}/api/v1',
      readAccess: () async => null,
      readRefresh: () async => null,
      writeTokens: (a, r) async {},
      clearTokens: () async {},
    ).cursor('p1', 12, 34, clientId: 'c1');

    expect(seen.map((u) => u.path), ['/api/v1/projects/p1/cursor']);
    expect(seen.single.queryParameters, {'client': 'c1'});
  });
}
