# toiconfont

builds an icon font from an IcoMoon compatible selection.json file

## Usage

```sh
# woff2 + woff + stylesheet
toiconfont -in selection.json -woff2 icons.woff2 -woff icons.woff -css icons.css

# set the font URLs written into the stylesheet
toiconfont -woff2 static/icons.woff2 -woff static/icons.woff -css icons.css \
  -css-woff2-url /static/icons.woff2 -css-url /static/icons.woff

# plain TrueType
toiconfont -ttf icons.ttf
```

| Flag | |
| --- | --- |
| `-in` | selection.json to read (default `selection.json`) |
| `-woff2` | write the WOFF2 font to this file |
| `-woff` | write the WOFF font to this file |
| `-ttf` | write the TrueType font to this file |
| `-css` | write the stylesheet to this file, needs `-woff2` or `-woff` |
| `-css-woff2-url` | WOFF2 URL used in the stylesheet (default: the WOFF2 file name) |
| `-css-url` | WOFF URL used in the stylesheet (default: the WOFF file name) |
| `-css-font-display` | `font-display` value used in the stylesheet: `auto`, `block`, `swap`, `fallback` or `optional` |
| `-v` | print version |

## Stylesheet

The stylesheet has a `@font-face` rule, a base rule for the class prefix,
and one class per icon:

```css
.icon-heart:before {
  content: "\e900";
}
```

The `@font-face` rule lists the WOFF2 font first, then the WOFF font, for
whichever of the two were written. Each font URL gets a hash of the font's
contents appended as a query string (`icons.woff2?1a2b3c4d`).

## selection.json

These fields are read, everything else is ignored:

| Field | Default |
| --- | --- |
| `height` | `1024` |
| `icons[].icon.paths` | |
| `icons[].icon.width` | `height` |
| `icons[].properties.name` | required, must be unique |
| `icons[].properties.code` | required, must be unique |
| `preferences.fontPref.prefix` | `icon-` |
| `preferences.fontPref.metadata.fontFamily` | `icomoon` |
| `preferences.fontPref.metrics.emSize` | `height` |
| `preferences.fontPref.metrics.baseline` | `6.25` |

Every path in an icon is merged into one glyph.
