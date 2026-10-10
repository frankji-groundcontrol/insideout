# Working-version autosave is promised but not built

Open, recorded 2026-10-11 (review-3 wave 4, finding FULL-03; slice 9 of the
[promise-gap set](2026-10-11-product-promise-gaps.md)).

## What PRODUCT.md promises

> "**Working version:** autosaved, mutable work; autosave is not a formal
> product version." (PRODUCT.md:201-202)

## What exists today

- No autosave or debounce machinery anywhere in `client/lib`
  (`rg -i "autosave|debounce"` returns nothing, checked 2026-10-11); saves
  are manual `guardWrite` actions
  (`client/lib/features/prd/prd_page.dart:117,135,185`).

## Failure

A user typing into the working version loses everything since the last
explicit save when the tab closes — the product's own teaching ("a workable
version beats an endless idea") starts from a draft that cannot survive a
mistake.

## Slice order

9 of the set — smallest and client-local; last because it touches no other
slice's contracts.

## Fix prompt

Red first: a widget test that edits section text, re-mounts the page
without pressing save, and still finds the text fails. Smallest slice:
debounce the working sections and persist the draft — client-side session
storage first, or against the existing revision endpoint on a timer;
never auto-Commit: autosave must not create a version row or touch the
versions stack (PRODUCT.md:201-202).
