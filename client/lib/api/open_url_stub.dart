/// Opens [url] in a new browser tab. Returns false when the platform has no
/// browser to hand it to, so callers keep an inline fallback (copy the
/// address and say so).
bool openExternalUrl(String url) => false;
