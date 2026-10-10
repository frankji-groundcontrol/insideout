import 'zh_account.dart';
import 'zh_prd.dart';
import 'zh_shell.dart';
import 'zh_workspace.dart';

const zhCN = <String, String>{
  ...zhShell,
  ...zhAccount,
  ...zhWorkspace,
  ...zhPrd,
};

const zhCoachSuggest = <String, List<String>>{
  'clarify': ['这到底解决了什么问题？', '目标用户是谁？', '它和现有方案相比有何不同？'],
  'draft': ['起草目标与成功指标。', '写一写核心用户故事。', '列出功能需求。'],
  'critique': ['这里哪些地方含糊或缺失？', '挑战一下范围——哪些该列为非目标？', '有哪些我没看到的风险？'],
  'finalize': ['把所有章节的语言收紧一遍。', '用一段话总结这份 PRD。', '这份 PRD 可以交给团队做了吗？'],
};
