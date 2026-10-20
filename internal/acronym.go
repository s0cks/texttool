package internal

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type AcronymMode string

const (
	SpellAcronymsMode  AcronymMode = "spell"
	ExpandAcronymsMode AcronymMode = "expand"
	KeepAcronymsMode   AcronymMode = "keep"
)

const DefaultAcronymMode = SpellAcronymsMode

func getAcronymModes() []string {
	return []string{
		string(SpellAcronymsMode),
		string(ExpandAcronymsMode),
		string(KeepAcronymsMode),
	}
}

func GetAcronymMode() AcronymMode {
	a := viper.GetString("acronyms")
	for _, b := range getAcronymModes() {
		if a == b {
			return AcronymMode(b)
		}
	}

	return DefaultAcronymMode
}

func AddAcryonymFlags(cmd *cobra.Command) {
	cmd.Flags().String("acronyms", string(DefaultAcronymMode), fmt.Sprintf("Specify how to handle acronyms. Valid values: %s", strings.Join(getAcronymModes(), ", ")))
}
