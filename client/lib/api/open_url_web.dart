import 'package:web/web.dart' as web;

/// Opens [url] in a new tab that gets no `window.opener` and no referrer.
/// An anchor click, not `window.open`: it is a navigation the user's own tap
/// started, so a popup blocker leaves it alone, and the answer is not lost to
/// `noopener` making `open` return null. Only https addresses are opened; a
/// `javascript:` or `data:` URL handed in later is refused (false), and the
/// caller falls back to copying it.
bool openExternalUrl(String url) {
  if (Uri.tryParse(url)?.scheme != 'https') return false;
  final anchor = web.HTMLAnchorElement()
    ..href = url
    ..target = '_blank'
    ..rel = 'noopener noreferrer';
  web.document.body?.appendChild(anchor);
  anchor.click();
  anchor.remove();
  return true;
}
