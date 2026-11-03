# Fonts

Press Start 2P, Silkscreen and VT323 come from `@fontsource`, Latin only.

`fusion-pixel-{zh,ko,ja}.woff2` are [Fusion Pixel](https://github.com/TakWolf/fusion-pixel-font)
12px proportional, release 2026.10.25 (SIL OFL, see `FusionPixel.OFL.txt`), trimmed to the
characters in `../i18n.js` plus ASCII. The full fonts are 700 KB each.

**A new translation can need a character these files don't have.** It would then show in the
system font. To re-trim, collect the language's characters (`allText(lang)` in `i18n.js`, plus
ASCII and the language names in the picker) and run:

```sh
pyftsubset fusion-pixel-12px-proportional-ja.otf.woff2 --text-file=chars-ja.txt \
  --layout-features='*' --flavor=woff2 --output-file=fusion-pixel-ja.woff2
```
