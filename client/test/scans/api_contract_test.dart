import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// TEST-04: nothing used to fail when the server's routes and the client's
/// called paths drifted apart — every client suite fakes the transport, and
/// the server's own tests never promised the old names to a client. This
/// pins the two sides together at the path/method level: every HTTP path
/// literal under lib/api/ must resolve to a route the server actually
/// registers. Cross-tree imports are impossible, so each side is read as
/// source: the client's `_dio`/`ApiCall`/`$baseUrl` literals on one side,
/// and the server's `HandleFunc` registration calls on the other — the same
/// inventory server/internal/api's auth scan builds. Rename a route on
/// either side and this goes red.
void main() {
  test('every client API path resolves to a registered server route', () {
    final routes = serverRoutes();
    final calls = clientCalls();

    // A scan that finds almost nothing would pass vacuously.
    expect(routes, isNotEmpty);
    expect(
      calls.length,
      greaterThanOrEqualTo(40),
      reason: 'found only ${calls.length} client calls; the scan is broken',
    );

    final unresolved = <String>[];
    for (final call in calls) {
      if (!routes.any((r) => r.matches(call))) {
        unresolved.add(
          '${call.method ?? '(any)'} ${call.path}  (${call.origin})',
        );
      }
    }
    expect(
      unresolved,
      isEmpty,
      reason:
          'Client calls with no registered server route (the contract '
          'drifted; align the name or the test):\n'
          '${unresolved.join('\n')}',
    );
  });
}

/// The server's `/api/v1` prefix every client path hangs off — pinned here
/// so a move of the public prefix reds this suite too.
const apiPrefix = '/api/v1';

/// Where the server's route registrations live, relative to the client
/// package root (both check out together; CI runs the client job on the
/// full repo).
const serverApiDir = '../server/internal/api';

class ServerRoute {
  ServerRoute(this.method, this.segments);

  /// Uppercase HTTP method, or null when the pattern names no method (any).
  final String? method;
  final List<String> segments;

  bool matches(ClientCall call) {
    if (method != null && call.method != null && method != call.method) {
      return false;
    }
    if (segments.length != call.segments.length) return false;
    for (var i = 0; i < segments.length; i++) {
      final server = segments[i];
      final client = call.segments[i];
      final serverWild = server.startsWith('{');
      final clientWild = client == '*';
      // A client `$id` segment is a value only a `{param}` slot can take;
      // two literals must agree exactly.
      if (serverWild != clientWild) return false;
      if (!serverWild && server != client) return false;
    }
    return true;
  }
}

class ClientCall {
  ClientCall(this.method, this.path, this.origin);

  final String? method; // null for the method-agnostic `$baseUrl` literals
  final String path;
  final String origin;
  late final segments = _segmentsOf(path);
}

/// Splits a path into matchable segments: query strings drop, and any
/// Dart-interpolated (`$var`) segment collapses to the `*` wildcard.
List<String> _segmentsOf(String path) => path
    .split('?')
    .first
    .split('/')
    .where((s) => s.isNotEmpty)
    .map((s) => s.contains(r'$') ? '*' : s)
    .toList();

/// Every HTTP path the client constructs under lib/api/. Three forms:
/// `_dio.<verb>('...')`, the `ApiCall('VERB', '...')` builders in
/// requests.dart, and the web-only `'$baseUrl/...'` fetch literals (their
/// native twins carry the method; they are checked method-agnostically).
List<ClientCall> clientCalls() {
  final calls = <ClientCall>[];
  final dioCall = RegExp(
    r"_dio\.(get|post|put|patch|delete)(?:<[^>]*>)?\(\s*'([^']*)'",
  );
  final apiCall = RegExp(
    r"ApiCall\(\s*'(GET|POST|PUT|PATCH|DELETE)',\s*'([^']*)'",
  );
  final baseUrlLiteral = RegExp(r"'\$base(?:Url|_url)/([^']*)'");
  for (final file in _dartFilesUnder('lib/api')) {
    final src = file.readAsStringSync();
    for (final m in dioCall.allMatches(src)) {
      calls.add(ClientCall(m.group(1)!.toUpperCase(), m.group(2)!, _rel(file)));
    }
    for (final m in apiCall.allMatches(src)) {
      calls.add(ClientCall(m.group(1)!, m.group(2)!, _rel(file)));
    }
    for (final m in baseUrlLiteral.allMatches(src)) {
      calls.add(ClientCall(null, '/${m.group(1)}', _rel(file)));
    }
  }
  return calls;
}

/// The routes the server registers: each `HandleFunc`/`Handle` first
/// argument, folded from string literals and same-file string consts, then
/// stripped of [apiPrefix]. A pattern this scan cannot fold fails the test
/// rather than slipping past it (the api package's auth scan holds the same
/// line).
List<ServerRoute> serverRoutes() {
  final dir = Directory(serverApiDir);
  if (!dir.existsSync()) {
    fail(
      'server api sources not found at $serverApiDir — this pin needs '
      'the server tree checked out beside the client',
    );
  }
  final routes = <ServerRoute>[];
  final handleCall = RegExp(r'\.(?:HandleFunc|Handle)\(([^,]*)');
  for (final file in _goFilesUnder(serverApiDir)) {
    final src = file.readAsStringSync();
    final consts = <String, String>{};
    for (final m in RegExp(
      r'(?:const|var)\s+(\w+)\s*=\s*"([^"]*)"',
    ).allMatches(src)) {
      consts[m.group(1)!] = m.group(2)!;
    }
    for (final m in handleCall.allMatches(src)) {
      final pattern = _foldArg(m.group(1)!, consts, _rel(file));
      final parts = pattern.split(RegExp(r'\s+'));
      final path = parts.length == 2 ? parts[1] : pattern;
      if (!path.startsWith(apiPrefix)) continue; // e.g. /healthz: not ours
      routes.add(
        ServerRoute(
          parts.length == 2 ? parts[0].toUpperCase() : null,
          path
              .substring(apiPrefix.length)
              .split('/')
              .where((s) => s.isNotEmpty)
              .toList(),
        ),
      );
    }
  }
  expect(
    routes.length,
    greaterThanOrEqualTo(60),
    reason: 'parsed only ${routes.length} server routes; the scan is broken',
  );
  return routes;
}

/// Folds a Go pattern argument — `"GET /x"`, `someConst`, or `"POST " +
/// webhookPath` — into one string, resolving identifiers through the
/// file's string consts.
String _foldArg(String arg, Map<String, String> consts, String origin) {
  final parts = RegExp(r'"([^"]*)"|([A-Za-z_]\w*)').allMatches(arg);
  final out = StringBuffer();
  for (final p in parts) {
    final lit = p.group(1);
    if (lit != null) {
      out.write(lit);
    } else {
      final ident = p.group(2)!;
      expect(
        consts,
        contains(ident),
        reason:
            '$origin: HandleFunc pattern uses $ident, which is not a '
            'string const this scan can fold; inline the literal or extend it',
      );
      out.write(consts[ident]);
    }
  }
  return out.toString();
}

List<File> _dartFilesUnder(String path) => Directory(path)
    .listSync(recursive: true)
    .whereType<File>()
    .where((f) => f.path.endsWith('.dart'))
    .toList();

List<File> _goFilesUnder(String path) => Directory(path)
    .listSync(recursive: true)
    .whereType<File>()
    .where((f) => f.path.endsWith('.go') && !f.path.endsWith('_test.go'))
    .toList();

String _rel(File f) => f.path.split(Platform.pathSeparator).skip(2).join('/');
