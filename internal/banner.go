package internal

import (
	"strings"
	"thunder/protocol"
)

var (
	banner = "\n" +
		"  _____ _                     _           \n" +
		" |_   _| |__  _   _ _ __   __| | ___ _ __ \n" +
		"   | | | '_ \\| | | | '_ \\ / _` |/ _ \\ '__|\n" +
		"   | | | | | | |_| | | | | (_| |  __/ |   \n" +
		"   |_| |_| |_|\\__,_|_| |_|\\__,_|\\___|_|   \n" +
		"                                          \n"
)

func BannerString() string {
	sb := strings.Builder{}
	sb.WriteString(banner)
	sb.WriteString("Thunder :: Version:")
	sb.WriteString("\t")
	sb.WriteString(protocol.CurrentVersion)
	sb.WriteString("\n")
	return sb.String()
}