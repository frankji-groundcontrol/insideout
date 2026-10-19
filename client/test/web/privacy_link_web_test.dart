@TestOn('browser')
library;

import 'dart:js_interop';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/auth/auth_door.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:web/web.dart' as web;

import '../_block_test_helpers.dart';

// rv4-PUB-02: the privacy link in a browser, end to end. The VM suite
// (auth_door_privacy_link_test.dart) is compiled with the stub opener, so all
// it can see is the copy fallback. Here the real widget is tapped under
// Chrome, where the conditional import in auth_door.dart has to resolve to
// open_url_web.dart: the tap must click an https anchor aimed at
// privacyNoticeUrl and copy nothing. Run it with
// `flutter test --platform chrome test/web`.
//
// A capture listener on the document sees the anchor's click and cancels it
// (no real navigation), so the click is observed instead of performed.

Future<void> _pumpLink(WidgetTester tester) => tester.pumpWidget(
  host(
    const PrivacyNoticeLink(
      label: 'Privacy notice',
      copiedMessage: 'Link copied',
    ),
    dark: false,
  ),
);

void main() {
  testWidgets('a tap opens the notice in a new tab and copies nothing', (
    tester,
  ) async {
    final clicks = <({String href, String target})>[];
    void onClick(web.Event event) {
      final anchor = event.target as web.HTMLAnchorElement;
      clicks.add((href: anchor.href, target: anchor.target));
      event.preventDefault();
    }

    final listener = onClick.toJS;
    web.document.addEventListener('click', listener, true.toJS);
    addTearDown(
      () => web.document.removeEventListener('click', listener, true.toJS),
    );

    final copied = <String>[];
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        if (call.method == 'Clipboard.setData') {
          copied.add((call.arguments as Map)['text'] as String);
        }
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );

    await _pumpLink(tester);
    await tester.tap(find.widgetWithText(BlockButton, 'Privacy notice'));
    await tester.pump();

    expect(clicks, hasLength(1), reason: 'the tap clicked one anchor');
    expect(clicks.single.href, privacyNoticeUrl);
    expect(clicks.single.target, '_blank');
    expect(copied, isEmpty, reason: 'the address was not copied');
    expect(find.text('Link copied'), findsNothing);
    expect(find.text('Privacy notice'), findsOneWidget);
  });

  testWidgets('in a browser the glyph says the link opens a tab', (
    tester,
  ) async {
    await _pumpLink(tester);

    expect(find.byIcon(Icons.open_in_new), findsOneWidget);
    expect(find.byIcon(Icons.link), findsNothing);
  });
}
