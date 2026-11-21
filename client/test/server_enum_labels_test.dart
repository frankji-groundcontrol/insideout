import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/prd/coach_column.dart';
import 'package:insideout/features/project/project_leaf_evidence.dart';
import 'package:insideout/session/appearance.dart';
import 'package:provider/provider.dart';

// rv5-FE-10: the coach speaker and the evidence kind are server enums; a
// Chinese reader sees them in Chinese, and an unknown value stays as sent.
void main() {
  final zh = Appearance(store: MemorySettingsStore(), locale: 'zh-CN');

  test('coach roles read in the active language', () {
    expect(coachRoleLabel(zh.t, 'assistant'), '教练');
    expect(coachRoleLabel(zh.t, 'user'), '你');
    expect(coachRoleLabel(zh.t, 'system'), 'system');
  });

  testWidgets('evidence rows name their kind in the active language', (
    tester,
  ) async {
    await tester.pumpWidget(
      ChangeNotifierProvider.value(
        value: zh,
        child: MaterialApp(
          home: Scaffold(
            body: ProjectLeafEvidence(
              future: Future.value((
                leaves: [
                  (
                    title: 'Ship v1',
                    rows: [
                      {'kind': 'release', 'detail': 'deploy succeeded'},
                      {'kind': 'webhook_x', 'detail': 'raw'},
                    ],
                  ),
                ],
                more: 0,
              )),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('发布: deploy succeeded'), findsOneWidget);
    expect(find.text('webhook_x: raw'), findsOneWidget);
  });
}
