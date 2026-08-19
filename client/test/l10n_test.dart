import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/l10n/en.dart';
import 'package:insideout/l10n/l10n.dart';
import 'package:insideout/l10n/zh.dart';

void main() {
  test('default locale matches Nuxt zh-CN', () {
    expect(defaultLocale, 'zh-CN');
    expect(lookup(defaultLocale, 'login.title'), '登录');
    expect(lookup(defaultLocale, 'coach.noConversation'), '这份 PRD 还没有关联教练对话。');
  });

  test('en-US and zh-CN catalogs share the same keys', () {
    expect(enUS.keys.toSet(), zhCN.keys.toSet());
  });

  test('interpolates placeholders the same way Nuxt does', () {
    expect(lookup('en-US', 'workspace.inviteCode', {'code': 'AB12CD'}), 'Invite code: AB12CD');
    expect(lookup('zh-CN', 'coach.rateLimited', {'seconds': 9}), '请求过于频繁，9 秒后重试...');
  });

  test('stage suggestions come from the Nuxt coach.suggest lists', () {
    expect(coachSuggestions('en-US', 'clarify'), [
      'What problem is this really solving?',
      'Who are the target users?',
      'What makes this different from existing options?',
    ]);
    expect(coachSuggestions('zh-CN', 'draft'), hasLength(3));
    expect(coachSuggestions('en-US', 'unknown'), isEmpty);
  });
}
