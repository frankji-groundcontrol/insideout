@TestOn('browser')
library;

import 'dart:js_interop';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/open_url_web.dart';
import 'package:web/web.dart' as web;

// rv4-PUB-02: the browser half of the privacy link. `flutter test` on the VM
// gets the stub that returns false (auth_door_privacy_link_test.dart covers
// the copy fallback that follows), and `flutter build web` only compiles this
// file, so it runs in Chrome (`flutter test --platform chrome test/web`).
//
// A capture listener on the document sees the anchor's click and cancels it
// (no real navigation), so each click is observed instead of performed.

void main() {
  late List<({String href, String target, String rel, bool attached})> clicks;
  late List<web.HTMLAnchorElement> anchors;

  setUp(() {
    clicks = [];
    anchors = [];
    void onClick(web.Event event) {
      final anchor = event.target as web.HTMLAnchorElement;
      anchors.add(anchor);
      clicks.add((
        href: anchor.href,
        target: anchor.target,
        rel: anchor.rel,
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

  test('opens an https address in a new tab with no opener', () {
    expect(openExternalUrl('https://example.com/PRIVACY.md'), isTrue);

    expect(clicks, hasLength(1));
    final click = clicks.single;
    expect(click.href, 'https://example.com/PRIVACY.md');
    expect(click.target, '_blank');
    expect(
      click.rel.split(RegExp(r'\s+')),
      containsAll(['noopener', 'noreferrer']),
    );
    expect(click.attached, isTrue, reason: 'clicked while in the document');
    expect(anchors.single.isConnected, isFalse, reason: 'removed afterwards');
  });

  test('refuses every scheme but https', () {
    for (final url in const [
      'javascript:alert(1)',
      'data:text/html,hello',
      'http://example.com/PRIVACY.md',
      '/PRIVACY.md',
      '',
    ]) {
      expect(openExternalUrl(url), isFalse, reason: url);
    }
    expect(clicks, isEmpty, reason: 'nothing was clicked');
  });
}
