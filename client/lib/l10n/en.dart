import 'en_account.dart';
import 'en_prd.dart';
import 'en_shell.dart';
import 'en_workspace.dart';

const enUS = <String, String>{
  ...enShell,
  ...enAccount,
  ...enWorkspace,
  ...enPrd,
};

const enCoachSuggest = <String, List<String>>{
  'clarify': [
    'What problem is this really solving?',
    'Who are the target users?',
    'What makes this different from existing options?',
  ],
  'draft': [
    'Draft the goals and success metrics.',
    'Write the core user stories.',
    'List the functional requirements.',
  ],
  'critique': [
    'What is vague or missing here?',
    'Challenge the scope — what should be a non-goal?',
    'What risks am I not seeing?',
  ],
  'finalize': [
    'Tighten the language across all sections.',
    'Summarize this PRD in one paragraph.',
    'Is this ready to hand off to the team?',
  ],
};
