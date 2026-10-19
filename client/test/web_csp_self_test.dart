import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// rv4-COMP-03: the web client must paint its first frame without a Google
/// host. A first load used to need three of them: www.gstatic.com (the
/// CanvasKit script and wasm), fonts.googleapis.com (a stylesheet CanvasKit
/// never reads) and fonts.gstatic.com (the Roboto and Noto fallback fonts the
/// engine downloads on its own). None of them is reachable from every
/// network, and a blocked CanvasKit is a page that never starts.
///
/// [engineFontHosts] is what may still leave the origin. The engine takes its
/// fallback fonts from `fontFallbackBaseUrl`, and only a custom
/// web/flutter_bootstrap.js can change that. Until that file lands this one
/// host stays in the policy; when it lands, empty the set, so the policy
/// is self-only and this test then forbids any host.
const engineFontHosts = <String>{'https://fonts.gstatic.com'};

void main() {
  final nginx = File('nginx.conf').readAsStringSync();

  test('web/index.html asks no other origin for anything', () {
    final html = File(
      'web/index.html',
    ).readAsStringSync().replaceAll(RegExp(r'<!--.*?-->', dotAll: true), '');
    final urls = RegExp(
      r'''(?:href|src)\s*=\s*["'](?:https?:)?//[^"']+''',
    ).allMatches(html).map((m) => m.group(0)!).toList();
    expect(urls, isEmpty, reason: 'index.html reaches off-origin: $urls');
  });

  test('the CSP lets scripts and styles come from this origin only', () {
    for (final name in ['default-src', 'script-src', 'style-src']) {
      final hosts = (_directives(nginx)[name] ?? const <String>[]).where(
        _isHost,
      );
      expect(hosts, isEmpty, reason: '$name lists $hosts');
    }
  });

  test('the CSP names no host beyond the engine font host', () {
    final hosts = {
      for (final sources in _directives(nginx).values)
        ...sources.where(_isHost),
    };
    expect(
      hosts.difference(engineFontHosts),
      isEmpty,
      reason: 'CSP hosts: $hosts, allowed: $engineFontHosts',
    );
  });

  test('every flutter build web bundles CanvasKit instead of the CDN', () {
    final builds = <String, String>{
      'client/Dockerfile': File('Dockerfile').readAsStringSync(),
      '.github/workflows/ci.yml': File(
        '../.github/workflows/ci.yml',
      ).readAsStringSync(),
    };
    for (final entry in builds.entries) {
      final lines = entry.value
          .split('\n')
          .where((l) => l.contains('flutter build web'))
          .toList();
      expect(lines, isNotEmpty, reason: '${entry.key} has no web build');
      for (final line in lines) {
        expect(
          line,
          contains('--no-web-resources-cdn'),
          reason: '${entry.key}: $line',
        );
      }
    }
  });

  test('the whole engine fallback table ships in web/fonts, with licenses', () {
    // The engine's table (Flutter 3.44) names 724 files in 144 families, and
    // it fetches a default Roboto besides: 725 files, 145 families. A file
    // that is missing is a 404 the engine retries in a tight loop. Which
    // family it asks for depends on the browser language (zh-TW asks for Noto
    // Sans TC, ja for JP, ko for KR) and on user text, so the mirror is the
    // whole table, not the slice this UI usually needs.
    final root = Directory('web/fonts/s');
    expect(root.existsSync(), isTrue, reason: 'web/fonts/s is missing');
    expect(
      File(
        'web/fonts/s/roboto/v32/KFOmCnqEu92Fr1Me4GZLCzYlKw.woff2',
      ).existsSync(),
      isTrue,
      reason: 'the default Roboto is missing',
    );
    final files = root.listSync(recursive: true).whereType<File>().toList();
    expect(files.length, 725);
    final families = <String>{};
    for (final f in files) {
      final raf = f.openSync();
      try {
        expect(
          String.fromCharCodes(raf.readSync(4)),
          'wOF2',
          reason: '${f.path} is not a woff2 file',
        );
      } finally {
        raf.closeSync();
      }
      families.add(
        f.path.replaceAll(r'\', '/').split('/')[3],
      ); // web/fonts/s/<family>/..
    }
    expect(families.length, 145);
    for (final family in families) {
      final license = File('web/fonts/licenses/$family.txt');
      expect(license.existsSync(), isTrue, reason: 'no license for $family');
      expect(
        license.readAsStringSync(),
        contains('SIL OPEN FONT LICENSE Version 1.1'),
        reason: family,
      );
    }
  });

  test('SHA256SUMS lists exactly the font files that ship', () {
    final listed = File('web/fonts/SHA256SUMS')
        .readAsLinesSync()
        .where((l) => l.trim().isNotEmpty)
        .map((l) => l.trim().split(RegExp(r'\s+')).last)
        .toSet();
    final shipped = Directory('web/fonts/s')
        .listSync(recursive: true)
        .whereType<File>()
        .map((f) => f.path.replaceAll(r'\', '/').replaceFirst('web/fonts/', ''))
        .toSet();
    expect(listed.difference(shipped), isEmpty, reason: 'listed, not shipped');
    expect(shipped.difference(listed), isEmpty, reason: 'shipped, not listed');
  });

  test('a font that is not mirrored is a 404, never the SPA shell', () {
    final block = RegExp(r'location /fonts/ \{([^}]*)\}').firstMatch(nginx);
    expect(block, isNotNull, reason: 'nginx.conf has no `location /fonts/`');
    final body = block!.group(1)!;
    expect(body, contains(r'try_files $uri =404;'));
    // One add_header inside a location replaces the server-level set there,
    // and the CSP would vanish from the font responses.
    expect(body, isNot(contains('add_header')));
  });
}

bool _isHost(String source) => RegExp(r'^https?://').hasMatch(source);

/// The `add_header Content-Security-Policy` line as {directive: [sources]}.
Map<String, List<String>> _directives(String nginx) {
  final m = RegExp(
    r'add_header Content-Security-Policy "([^"]+)"',
  ).firstMatch(nginx);
  expect(m, isNotNull, reason: 'nginx.conf has no Content-Security-Policy');
  final out = <String, List<String>>{};
  for (final part in m!.group(1)!.split(';')) {
    final words = part.trim().split(RegExp(r'\s+'));
    if (words.first.isNotEmpty) out[words.first] = words.skip(1).toList();
  }
  return out;
}
