package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var manCmd = &cobra.Command{
	Use:    "man",
	Short:  "Generate man pages",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("dir")
		return doc.GenManTree(rootCmd, nil, dir)
	},
}

func init() {
	manCmd.Flags().String("dir", ".", "output directory")
	rootCmd.AddCommand(manCmd)
}
