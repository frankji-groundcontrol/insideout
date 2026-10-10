@TestOn('browser')
library;

import 'dart:js_interop';
import 'dart:js_interop_unsafe';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/download_web.dart';
import 'package:web/web.dart' as web;

// TEST-11: the browser save path under the export page: an object URL, a
// clicked anchor, a revoke. `flutter test` on the VM gets the stub that
// returns false, and `flutter build web` only compiles this, so it runs in
// Chrome (`flutter test --platform chrome test/web`).
//
// A capture listener on the document sees the anchor's click and cancels it
// (no real download), and URL.revokeObjectURL and window.open are replaced for
// the test, so each is observed instead of performed.

/// Puts [stub] at [object][name] for the rest of the test.
void _stub(JSObject object, String name, JSFunction stub) {
  final original = object[name];
  object[name] = stub;
  addTearDown(() => object[name] = original);
}

void main() {
  late List<String> revoked;
  late List<({String download, String href, bool attached})> clicks;
  late List<web.HTMLAnchorElement> anchors;

  setUp(() {
    revoked = [];
    clicks = [];
    anchors = [];
    // Recorded, not revoked: the test still has to read the blob.
    void revoke(JSString url) => revoked.add(url.toDart);
    _stub(globalContext['URL'] as JSObject, 'revokeObjectURL', revoke.toJS);

    void onClick(web.Event event) {
      final anchor = event.target as web.HTMLAnchorElement;
      anchors.add(anchor);
      clicks.add((
        download: anchor.download,
        href: anchor.href,
        attached: anchor.isConnected,
      ));
      event.preventDefault();
    }

    final listener = onClick.toJS;
    web.document.addEventListener('click', listener, true.toJS);
    addTearDown(
      () => web.document.removeEventListener('click', listener, true.toJS),
    );
  });

  test('saves the text as a blob download named by the caller', () async {
    expect(
      downloadTextFile('notes.md', 'hello', mime: 'text/markdown'),
      isTrue,
    );

    expect(clicks, hasLength(1));
    final click = clicks.single;
    expect(click.download, 'notes.md');
    expect(click.href, startsWith('blob:'));
    expect(click.attached, isTrue, reason: 'clicked while in the document');
    expect(anchors.single.isConnected, isFalse, reason: 'removed afterwards');
    expect(revoked, [click.href]);

    final res = await web.window.fetch(click.href.toJS).toDart;
    expect((await res.text().toDart).toDart, 'hello');
    expect(res.headers.get('content-type'), 'text/markdown');
  });

  test('openTextInNewTab says so when the browser blocks the popup', () {
    final opened = <(String, String)>[];
    JSAny? open(JSString url, JSString target) {
      opened.add((url.toDart, target.toDart));
      return null;
    }

    _stub(globalContext, 'open', open.toJS);

    expect(openTextInNewTab('<p>hello</p>', mime: 'text/html'), isFalse);
    expect(opened.single.$1, startsWith('blob:'));
    expect(opened.single.$2, '_blank');
  });

  test('openTextInNewTab leaves the object URL alive for the new tab', () {
    JSAny? open(JSString url, JSString target) => web.document.body;
    _stub(globalContext, 'open', open.toJS);

    expect(openTextInNewTab('hello'), isTrue);
    // Revoking now would cancel the tab's own load of the URL.
    expect(revoked, isEmpty);
  });
}
