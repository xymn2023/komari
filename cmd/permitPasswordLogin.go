package cmd

import (
	"os"

	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/database/models"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var PermitPasswordLoginCmd = &cobra.Command{
	Use:   "permit-login",
	Short: "Force permit password login",
	Long:  `Force permit password login`,
	Run: func(cmd *cobra.Command, args []string) {
		db := dbcore.GetDBInstance()
		err := db.Transaction(func(tx *gorm.DB) error {
			return tx.Model(&models.Config{}).Where("id = ?", 1).
				Update("disable_password_login", false).Error
		})
		if err != nil {
			cmd.Println("Error:", err)
			os.Exit(1)
		}
		cmd.Println("Password login has been permitted.")
		cmd.Println("Please restart the server to apply the changes.")
	},
}

func init() {
	RootCmd.AddCommand(PermitPasswordLoginCmd)
}
