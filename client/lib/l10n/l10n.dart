import 'en.dart';
import 'zh.dart';

const defaultLocale = 'zh-CN';

String lookup(String locale, String key, [Map<String, Object?>? args]) {
  final catalog = locale == 'en-US' ? enUS : zhCN;
  var text = catalog[key] ?? enUS[key] ?? key;
  if (args != null) {
    for (final entry in args.entries) {
      text = text.replaceAll('{${entry.key}}', '${entry.value}');
    }
  }
  return text;
}

List<String> coachSuggestions(String locale, String stage) {
  final table = locale == 'en-US' ? enCoachSuggest : zhCoachSuggest;
  return List<String>.from(table[stage] ?? const <String>[]);
}
