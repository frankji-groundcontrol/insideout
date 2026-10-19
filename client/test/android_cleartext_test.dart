import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// rv4-COMP-06: a release APK must not allow cleartext HTTP. The main manifest
/// set `android:usesCleartextTraffic="true"`, and debug, profile and release
/// all merge it, so every build type accepted plain http to any host. Now main
/// says false, and the debug and profile source sets (never shipped) reach a
/// local dev server through a network security config that allows cleartext
/// for loopback names only.
///
/// This reads the files, so it needs no Android toolchain. It cannot show the
/// merged manifest of a built APK: that takes the Android SDK
/// (`flutter build apk`, then `apkanalyzer manifest print`).
const loopbackHosts = <String>{'localhost', '127.0.0.1', '10.0.2.2'};
const srcDir = 'android/app/src';
const devSourceSets = ['debug', 'profile'];

void main() {
  test('the main manifest, which every build type merges, says false', () {
    final main = _read('$srcDir/main/AndroidManifest.xml');
    expect(_attr(main, 'application', 'android:usesCleartextTraffic'), 'false');
  });

  test('no manifest in any source set turns cleartext on', () {
    final manifests = Directory(srcDir)
        .listSync(recursive: true)
        .whereType<File>()
        .where((f) => f.path.endsWith('AndroidManifest.xml'))
        .toList();
    expect(manifests.length, greaterThanOrEqualTo(3));
    for (final f in manifests) {
      expect(
        _attr(_read(f.path), 'application', 'android:usesCleartextTraffic'),
        isNot('true'),
        reason: f.path,
      );
    }
  });

  test('release carries no network security config', () {
    final main = _read('$srcDir/main/AndroidManifest.xml');
    expect(_attr(main, 'application', 'android:networkSecurityConfig'), isNull);
    expect(
      File('$srcDir/main/res/xml/network_security_config.xml').existsSync(),
      isFalse,
      reason: 'a config under src/main would ship in release',
    );
  });

  for (final set in devSourceSets) {
    test('$set points at a network security config that exists', () {
      final manifest = _read('$srcDir/$set/AndroidManifest.xml');
      expect(
        _attr(manifest, 'application', 'android:networkSecurityConfig'),
        '@xml/network_security_config',
      );
      expect(
        File('$srcDir/$set/res/xml/network_security_config.xml').existsSync(),
        isTrue,
      );
    });

    test('$set allows cleartext for loopback hosts only', () {
      final path = '$srcDir/$set/res/xml/network_security_config.xml';
      expect(File(path).existsSync(), isTrue, reason: '$path is missing');
      final xml = _read(path);
      expect(
        _attr(xml, 'base-config', 'cleartextTrafficPermitted'),
        'false',
        reason: 'everything not listed below must stay on https',
      );
      final blocks = RegExp(
        r'<domain-config\b([^>]*)>(.*?)</domain-config>',
        dotAll: true,
      ).allMatches(xml).toList();
      final allowing = blocks.where(
        (b) => b.group(1)!.contains('cleartextTrafficPermitted="true"'),
      );
      // Every `true` in the file sits on a domain-config, none on base-config.
      expect(
        RegExp(r'cleartextTrafficPermitted="true"').allMatches(xml).length,
        allowing.length,
      );
      expect(allowing, isNotEmpty, reason: 'the dev server needs http');
      for (final block in allowing) {
        final hosts = RegExp(
          r'<domain\b[^>]*>\s*([^<\s]+)\s*</domain>',
        ).allMatches(block.group(2)!).map((m) => m.group(1)!).toSet();
        expect(hosts, isNotEmpty);
        expect(
          hosts.difference(loopbackHosts),
          isEmpty,
          reason: 'cleartext allowed beyond loopback: $hosts',
        );
      }
    });
  }
}

/// The file's text without XML comments (their prose may name the attributes).
String _read(String path) {
  final file = File(path);
  expect(file.existsSync(), isTrue, reason: '$path is missing');
  return file.readAsStringSync().replaceAll(
    RegExp(r'<!--.*?-->', dotAll: true),
    '',
  );
}

/// Attribute [name] of the first `<tag ...>` start tag in [xml], or null.
String? _attr(String xml, String tag, String name) {
  final start = RegExp('<$tag\\b([^>]*)>').firstMatch(xml);
  if (start == null) return null;
  return RegExp(
    '$name\\s*=\\s*"([^"]*)"',
  ).firstMatch(start.group(1)!)?.group(1);
}
