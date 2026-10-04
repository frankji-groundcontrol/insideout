import 'dart:js_interop';

import 'package:web/web.dart' as web;

bool downloadTextFile(String filename, String text, {String mime = 'text/plain'}) {
  final url = _objectUrl(text, mime);
  final anchor = web.HTMLAnchorElement()
    ..href = url
    ..download = filename;
  web.document.body?.appendChild(anchor);
  anchor.click();
  anchor.remove();
  web.URL.revokeObjectURL(url);
  return true;
}

bool openTextInNewTab(String text, {String mime = 'text/plain'}) {
  final url = _objectUrl(text, mime);
  web.window.open(url, '_blank');
  // The new tab loads the object URL asynchronously; revoking immediately
  // would cancel that load, so give it a beat.
  web.window.setTimeout((() => web.URL.revokeObjectURL(url)).toJS, 60_000.toJS);
  return true;
}

String _objectUrl(String text, String mime) {
  final blob = web.Blob(
    <JSAny>[text.toJS].toJS,
    web.BlobPropertyBag(type: mime),
  );
  return web.URL.createObjectURL(blob);
}
