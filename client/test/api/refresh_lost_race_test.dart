import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/sse.dart';

/// rv4-FR-F112, client half. The server answers 401 to a spent one-time
/// refresh token it sees again inside its reuse grace, and the tab that lost
/// the race got the same answer as a revoked session: it cleared the shared
/// token store, which threw away the pair the winning tab had just stored. The
/// loser must look in the store for that pair and carry on with it (the Go
/// client does the same, apiclient renewalGrace). The server keeps its 401.

/// One tab: its client over a token store that another tab can write to, and a
/// server whose /auth/refresh has already been won by someone else.
class _Tab {
  _Tab(int port, this.hits) {
    client = ApiClient(
      baseUrl: 'http://127.0.0.1:$port/api/v1',
      readAccess: () async => access,
      readRefresh: () async => refresh,
      writeTokens: (a, r) async {
        access = a;
        refresh = r;
      },
      clearTokens: () async {
        cleared = true;
        access = null;
        refresh = null;
      },
    );
  }

  late final ApiClient client;
  final List<String> hits;
  String? access = 'access-1';
  String? refresh = 'refresh-1';
  bool cleared = false;

  int hitsFor(String line) => hits.where((h) => h == line).length;

  /// What the winning tab leaves in the shared store once its refresh lands.
  void winnerStores() {
    access = 'access-2';
    refresh = 'refresh-2';
  }
}

/// [winnerLandsAfter] is when the winner's pair reaches the store relative to
/// the loser's refused refresh; null means nobody rotated.
Future<_Tab> _tab({Duration? winnerLandsAfter, int refusal = 401}) async {
  final hits = <String>[];
  final server = await HttpServer.bind('127.0.0.1', 0);
  addTearDown(() => server.close(force: true));
  late final _Tab tab;
  server.listen((req) async {
    await utf8.decoder.bind(req).join();
    hits.add('${req.method} ${req.uri.path}');
    var status = 401;
    Object body = {'error': 'invalid or expired session'};
    if (req.uri.path == '/api/v1/auth/refresh') {
      final after = winnerLandsAfter;
      if (after == Duration.zero) tab.winnerStores();
      if (after != null && after > Duration.zero) {
        Timer(after, tab.winnerStores);
      }
      status = refusal;
      body = {'error': 'refresh token expired or already used'};
    } else if (req.uri.path == '/api/v1/me' &&
        req.headers.value(HttpHeaders.authorizationHeader) ==
            'Bearer access-2') {
      status = 200;
      body = {
        'id': 'user-1',
        'email': 'ada@example.com',
        'username': 'ada',
        'bio': '',
        'keywords': <String>[],
      };
    }
    req.response
      ..statusCode = status
      ..headers.contentType = ContentType.json
      ..write(jsonEncode(body));
    await req.response.close();
  });
  return tab = _Tab(server.port, hits);
}

void main() {
  group(
    'rv4-FR-F112: a refresh another tab won is not the end of the session',
    () {
      for (final refusal in [401, 403]) {
        test(
          'a $refusal with the winner\'s pair already stored adopts it',
          () async {
            final tab = await _tab(
              winnerLandsAfter: Duration.zero,
              refusal: refusal,
            );

            final me = await tab.client.me();

            expect(me.id, 'user-1');
            expect(
              tab.cleared,
              isFalse,
              reason: 'the loser wiped the shared store',
            );
            expect(tab.access, 'access-2');
            expect(tab.refresh, 'refresh-2');
            expect(
              tab.hitsFor('POST /api/v1/auth/refresh'),
              1,
              reason: 'adopting the stored pair spends no second rotation',
            );
            expect(tab.hitsFor('GET /api/v1/me'), 2);
          },
        );
      }

      test(
        'a winner whose pair lands a moment later is still adopted',
        () async {
          final tab = await _tab(
            winnerLandsAfter: const Duration(milliseconds: 120),
          );

          final me = await tab.client.me();

          expect(me.id, 'user-1');
          expect(tab.cleared, isFalse);
          expect(tab.refresh, 'refresh-2');
        },
      );

      test('a presence stream that lost the race reopens with the winner\'s '
          'bearer', () async {
        final tab = await _tab(winnerLandsAfter: Duration.zero);
        final bearers = <String?>[];
        tab.client.fetchAttemptForTests = (url, headers, {stop}) async {
          bearers.add(headers['Authorization']);
          if (bearers.length == 1) {
            return const SseAttempt(status: 401, chunks: Stream.empty());
          }
          return SseAttempt(
            status: 200,
            chunks: Stream.value(
              utf8.encode(
                'event: presence\n'
                'data: [{"sessionId":"s1","userId":"u1","name":"mover"}]\n\n',
              ),
            ),
          );
        };
        final seen = <String>[];

        await tab.client.presenceStream(
          'p1',
          'c1',
          (entries) => seen.addAll(entries.map((e) => e.name)),
        );

        expect(bearers, ['Bearer access-1', 'Bearer access-2']);
        expect(seen, ['mover']);
        expect(tab.cleared, isFalse);
      });

      test('with nothing new in the store the refusal stands and the session '
          'clears', () async {
        final tab = await _tab();

        await expectLater(
          tab.client.me(),
          throwsA(isA<ApiException>().having((e) => e.status, 'status', 401)),
        );

        expect(tab.cleared, isTrue);
        expect(tab.refresh, isNull);
      });
    },
  );
}
