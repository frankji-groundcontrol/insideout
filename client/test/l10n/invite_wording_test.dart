import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/l10n/en.dart';
import 'package:insideout/l10n/zh.dart';

// rv4-USER-10: a collaborator needs two unrelated codes and both were called
// "invite code" (in Chinese both were 邀请码): the email-bound invitation that
// lets an account be created, and the workspace code that lets a signed-in
// account into a team. Each now has one name per language, and a string that
// speaks of one never uses the other's word.
//
// The names are the decision this test pins; renaming one means changing it
// here and in the catalogs, nowhere else.
typedef _Words = ({
  Map<String, String> catalog,
  String workspaceCode, // what a team's join code is called
  String invitation, // what the account invitation is called
  String invitationStem, // the word stem every invitation string shares
});

final _languages = <String, _Words>{
  'en-US': (
    catalog: enUS,
    workspaceCode: 'workspace code',
    invitation: 'invitation code',
    invitationStem: 'invit',
  ),
  'zh-CN': (
    catalog: zhCN,
    workspaceCode: '工作区代码',
    invitation: '邀请码',
    invitationStem: '邀请',
  ),
};

// The strings that name the workspace code: the dialog field, the code shown
// on the workspace page, the hint above it and the empty-dashboard prompt.
const _workspaceCodeKeys = [
  'workspace.joinCodePlaceholder',
  'workspace.inviteCode',
  'workspace.inviteHint',
  'workspace.empty',
];

// The strings that name the account invitation: the register field, its error
// and the admin dialog's line about the code it created.
const _invitationKeys = [
  'register.invite',
  'register.inviteError',
  'admin.codeOnce',
];

// workspace.inviteAccountHint is the one workspace string that is about the
// invitation, on purpose: it tells a teammate to get an account first.
const _aboutTheInvitation = 'workspace.inviteAccountHint';

void main() {
  _languages.forEach((locale, words) {
    group(locale, () {
      test('the workspace code is called "${words.workspaceCode}"', () {
        for (final key in _workspaceCodeKeys) {
          expect(
            words.catalog[key]!.toLowerCase(),
            contains(words.workspaceCode),
            reason: key,
          );
        }
      });

      test('no workspace string borrows the invitation word', () {
        for (final entry in words.catalog.entries) {
          if (!entry.key.startsWith('workspace.') ||
              entry.key == _aboutTheInvitation) {
            continue;
          }
          expect(
            entry.value.toLowerCase(),
            isNot(contains(words.invitationStem)),
            reason: '${entry.key} is a workspace string: ${entry.value}',
          );
        }
      });

      test('the account invitation is called "${words.invitation}", only', () {
        for (final key in _invitationKeys) {
          final text = words.catalog[key]!.toLowerCase();
          expect(text, contains(words.invitation), reason: key);
          expect(text, isNot(contains(words.workspaceCode)), reason: key);
        }
      });
    });
  });
}
