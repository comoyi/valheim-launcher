package theme

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/comoyi/valheim-launcher/fonts"
	"image/color"
)

var CustomTheme = &Theme{}

type Theme struct {
}

func (t *Theme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (t *Theme) Font(style fyne.TextStyle) fyne.Resource {
	return fonts.DefaultFont
	//return theme.DefaultTheme().Font(style)
}

func (t *Theme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *Theme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
