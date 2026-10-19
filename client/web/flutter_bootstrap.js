{{flutter_js}}
{{flutter_build_config}}

// rv4-COMP-03: CanvasKit ignores CSS @font-face and downloads the fonts it
// paints with by URL (a default Roboto, plus the Noto slice for every glyph
// Roboto lacks). They default to fonts.gstatic.com; web/fonts/s mirrors every
// file the engine can ask for, so read them from there. Only this loader
// option can change it, and the CSP bars an inline script, hence this file.
_flutter.loader.load({
  config: {
    fontFallbackBaseUrl: "fonts/s/",
  },
  serviceWorkerSettings: {
    serviceWorkerVersion: {{flutter_service_worker_version}},
  },
});
