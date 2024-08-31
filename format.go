package main

import (
	"strconv"
	"strings"
)

const (
	modeFont             = 3
	modeBackground       = 4
	modeBrightFont       = 9
	modeBrightBackground = 10
)

const (
	colorBlack = iota
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorPurple
	colorCyan
	colorWhite
)

const (
	styleNormal          = 0
	styleBold            = 1
	styleFaint           = 2
	styleItalic          = 3
	styleUnderline       = 4
	styleBlink           = 5
	styleBlinkSlow       = 6
	styleInvert          = 7
	styleHidden          = 8
	styleStrikethrough   = 9
	styleDoubleUnderline = 21
	styleOverline        = 53
)

func colored(str string, color, mode, style int) string {
	var sb strings.Builder
	if style > 0 {
		sb.WriteString("\033[")
		sb.WriteString(strconv.Itoa(style))
		sb.WriteString("m")
	}
	sb.WriteString("\033[")
	sb.WriteString(strconv.Itoa(mode))
	sb.WriteString(strconv.Itoa(color))
	sb.WriteString("m")
	sb.WriteString(str)
	sb.WriteString("\033[0m") // reset
	return sb.String()
}

func red(str string) string {
	return colored(str, colorRed, modeFont, styleNormal)
}

func green(str string) string {
	return colored(str, colorGreen, modeFont, styleNormal)
}

func yellow(str string) string {
	return colored(str, colorYellow, modeFont, styleNormal)
}

func blue(str string) string {
	return colored(str, colorBlue, modeFont, styleNormal)
}

func purple(str string) string {
	return colored(str, colorPurple, modeFont, styleNormal)
}

func cyan(str string) string {
	return colored(str, colorCyan, modeFont, styleNormal)
}
