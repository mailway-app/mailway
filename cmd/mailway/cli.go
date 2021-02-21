package main

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	isLocalSetup bool

	rootCmd = &cobra.Command{
		Use:   "mailway",
		Short: "Mailway CLI",
	}
	setupCmd = &cobra.Command{
		Use:   "setup",
		Short: "Mailway instance setup",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setup(); err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}
	setupSecureSMTPCmd = &cobra.Command{
		Use:   "setup-secure-smtp",
		Short: "Mailway instance setup secure inbound SMTP",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupSecureSmtp(); err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}
	generateFrontlineConfigCmd = &cobra.Command{
		Use:   "generate-frontline-config",
		Short: "Mailway instance generate the frontline NGINX configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := generateFrontlineConf(); err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}
	newJWTCmd = &cobra.Command{
		Use:   "new-jwt",
		Short: "Mailway instance generate a new JWT token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := newJWT(); err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}
	restartCmd = &cobra.Command{
		Use:   "restart",
		Short: "Restart Mailway services",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			services("restart")
			return nil
		},
	}
	logsCmd = &cobra.Command{
		Use:   "logs",
		Short: "Display Mailway logs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			logs()
			return nil
		},
	}
	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Get Mailway services status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			services("restart")
			return nil
		},
	}
	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update Mailway services",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := update(); err != nil {
				return errors.Wrap(err, "could not update")
			}
			return nil
		},
	}
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Print Mailway configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			printConfig()
			return nil
		},
	}
	supervisorCmd = &cobra.Command{
		Use:   "supervisor",
		Short: "Run Mailway supervisor",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := supervise(); err != nil {
				return errors.Wrap(err, "failed to supervise")
			}
			return nil
		},
	}
	recoverCmd = &cobra.Command{
		Use:   "recover [file]",
		Short: "Run Mailway supervisor",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := recoverEmail(args[0]); err != nil {
				return errors.Wrap(err, "could not recover email")
			}
			return nil
		},
	}
)

func init() {
	setupCmd.Flags().BoolVar(&isLocalSetup, "local", false,
		"Don't connect with Mailway API, run in local mode")

	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(setupSecureSMTPCmd)
	rootCmd.AddCommand(generateFrontlineConfigCmd)
	rootCmd.AddCommand(newJWTCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(supervisorCmd)
	rootCmd.AddCommand(recoverCmd)
}
