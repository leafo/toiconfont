# toiconfont

builds an icon font from an IcoMoon compatible selection.json file

## Usage

```sh
# woff + stylesheet
toiconfont -in selection.json -woff icons.woff -css icons.css

# set the font URL written into the stylesheet
toiconfont -woff static/icons.woff -css icons.css -css-url /static/icons.woff

# plain TrueType
toiconfont -ttf icons.ttf
```

| Flag | |
| --- | --- |
| `-in` | selection.json to read (default `selection.json`) |
| `-woff` | write the WOFF font to this file |
| `-ttf` | write the TrueType font to this file |
| `-css` | write the stylesheet to this file, needs `-woff` |
| `-css-url` | font URL used in the stylesheet (default: the WOFF file name) |
| `-v` | print version |

## Stylesheet

The stylesheet has a `@font-face` rule, a base rule for the class prefix,
and one class per icon:

```css
.icon-heart:before {
  content: "\e900";
}
```

The font URL gets a hash of the font's contents appended as a query string
(`icons.woff?1a2b3c4d`).

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
