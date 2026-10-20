package internal

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type HandlesMode string

const (
	SayHandlesMode   HandlesMode = "say"
	StripHandlesMode HandlesMode = "strip"
	KeepHandlesMode  HandlesMode = "keep"
)

const DefaultHandlesMode HandlesMode = SayHandlesMode

func getAllHandlesModes() []string {
	return []string{
		string(SayHandlesMode),
		string(StripHandlesMode),
		string(KeepHandlesMode),
	}
}

func GetHandlesMode() HandlesMode {
	switch viper.GetString("handles") {
	case string(SayHandlesMode):
		return SayHandlesMode

	case string(StripHandlesMode):
		return StripHandlesMode

	case string(KeepHandlesMode):
		return KeepHandlesMode

	default:
		return DefaultHandlesMode
	}
}

func AddHandlesFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.String("handles", string(DefaultHandlesMode), fmt.Sprintf("@handles with --tts: %s", strings.Join(getAllHandlesModes(), ", ")))
}
