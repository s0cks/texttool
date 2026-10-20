package cmd

import (
	"fmt"

	"github.com/s0cks/texttool/internal"
	"github.com/spf13/cobra"
)

func init() {
	version := &cobra.Command{
		Use: "version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(internal.GetVersion())
			return nil
		},
	}
	RootCommand.AddCommand(version)
}
