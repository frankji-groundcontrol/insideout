import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:insideout/theme/block_chrome.dart';
import 'package:insideout/theme/block_modal.dart';
import 'package:insideout/theme/cobalt.dart';

import '_block_test_helpers.dart';

void main() {
  testWidgets(
    'showBlockModal: header, 2px divider, scrollable body, footer bar',
    (tester) async {
      await tester.pumpWidget(
        host(
          Builder(
            builder: (context) => Center(
              child: BlockButton(
                label: 'Open',
                onPressed: () => showBlockModal<void>(
                  context: context,
                  title: 'Modal title',
                  body: const Text('Modal body copy'),
                  footer: [
                    BlockButton(
                      label: 'Done',
                      variant: BlockButtonVariant.secondary,
                      onPressed: () => Navigator.of(context).pop(),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Open'));
      await tester.pumpAndSettle();

      expect(find.text('Modal title'), findsOneWidget);
      expect(find.byType(BlockIconButton), findsOneWidget);
      expect(find.byIcon(Icons.close), findsOneWidget);

      final divider = tester.widget<Divider>(find.byType(Divider));
      expect(divider.height, 2);
      expect(divider.thickness, 2);
      expect(divider.color, Cobalt.dark.strokeSubtle);

      expect(
        find.widgetWithText(SingleChildScrollView, 'Modal body copy'),
        findsOneWidget,
      );

      expect(find.text('Done'), findsOneWidget);
      final footerBar = tester
          .widgetList<Container>(
            find.ancestor(
              of: find.text('Done'),
              matching: find.byType(Container),
            ),
          )
          .map((w) => w.decoration)
          .whereType<BoxDecoration>()
          .firstWhere((d) => d.color == Cobalt.dark.surfaceSunken);
      expect(footerBar.color, Cobalt.dark.surfaceSunken);

      final panel = tester.getSize(
        find
            .ancestor(
              of: find.text('Modal title'),
              matching: find.byType(Material),
            )
            .first,
      );
      expect(panel.width, 512);
    },
  );

  testWidgets('showBlockConfirm resolves true on the danger confirm', (
    tester,
  ) async {
    bool? result;
    await tester.pumpWidget(
      host(
        Builder(
          builder: (context) => Center(
            child: BlockButton(
              label: 'Ask',
              onPressed: () async {
                result = await showBlockConfirm(
                  context: context,
                  title: 'Delete node?',
                  message: 'This cannot be undone.',
                  danger: true,
                );
              },
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Ask'));
    await tester.pumpAndSettle();
    expect(find.text('Delete node?'), findsOneWidget);
    expect(find.text('This cannot be undone.'), findsOneWidget);
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(result, isTrue);
    expect(find.text('Delete node?'), findsNothing);
  });
}
