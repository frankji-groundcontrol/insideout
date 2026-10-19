@TestOn('browser')
library;

import 'dart:js_interop';
import 'dart:js_interop_unsafe';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/web_fonts_web.dart';
import 'package:web/web.dart' as web;

// rv4-FR-F14, F215, F210, D01 (open residue R3-30): the web boot gate.
// main.dart holds the boot loading page until loadWebFonts() returns. The gate
// polls document.fonts.load() for CSS faces (Archivo, Noto Sans SC, Source
// Serif 4, Noto Serif SC), but rv4-COMP-03 took the only stylesheet that
// declared them out of web/index.html, and CanvasKit ignores CSS @font-face
// anyway. This page declares none of them, so load() can only resolve empty and
// the gate ran to its 10 s deadline on every cold start.
//
// Needs a browser (`flutter test --platform chrome test/web`): document.fonts
// does not exist on the VM, where the stub returns at once.

void main() {
  test('lifts the boot gate in well under the old 10 s deadline', () async {
    final watch = Stopwatch()..start();
    await loadWebFonts();
    expect(
      watch.elapsed,
      lessThan(const Duration(seconds: 2)),
      reason: 'no stylesheet declares a face, so nothing can arrive',
    );
  });

  test('asks document.fonts for no face that no stylesheet declares', () async {
    final fonts = web.document.fonts as JSObject;
    final original = fonts['load'] as JSFunction;
    final asked = <String>[];
    JSAny? record(JSString face, JSAny? text) {
      asked.add(face.toDart);
      return original.callAsFunction(fonts, face, text);
    }

    fonts['load'] = record.toJS;
    addTearDown(() => fonts['load'] = original);

    await loadWebFonts();
    expect(
      asked,
      isEmpty,
      reason: 'polled ${asked.length} times, first ${asked.take(2).toList()}',
    );
  });
}
