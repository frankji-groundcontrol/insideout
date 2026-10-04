/// Downloads [text] as a browser file save (FEAT-03). Returns false when
/// the platform has no download affordance, so callers keep their inline
/// fallback.
bool downloadTextFile(String filename, String text, {String mime = 'text/plain'}) => false;

/// Opens [text] rendered as [mime] in a new browser tab (the print path:
/// the browser's own print-to-PDF dialog is one keystroke away). Returns
/// false when unavailable.
bool openTextInNewTab(String text, {String mime = 'text/plain'}) => false;
