package main

import (
	"context"
	"fmt"
	"os"

	"github.com/glifio/invariants/singleton"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// The name of our config file, without the file extension because viper supports many different config file languages.
	defaultConfigFilename = "mainnet"

	// The environment variable prefix of all environment variables bound to our command line flags.
	// For example, --number is bound to GRAPH_NUMBER.
	envPrefix = "INVARIANTS"
)

var (
	// Used for flags.
	cfgFile string

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "invariants",
		Short: "Checks values from REST API against Lotus node values",
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "mainnet", "config file (default is ./mainnet.env)")

	viper.BindEnv("port")
	viper.BindEnv("chain_id")
	viper.BindEnv("lotus_archive_token")
	viper.BindEnv("lotus_archive_addr")
	viper.BindEnv("lotus_private_token")
	viper.BindEnv("lotus_private_addr")
	viper.BindEnv("postgres_conn")
	viper.BindEnv("tablename_suffix")
	viper.BindEnv("events_api")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		// log.Printf("config file from the flag %s\n", cfgFile)
		viper.AddConfigPath(".")
		viper.SetConfigName(cfgFile)
		viper.SetConfigType("env")
	} else {
		viper.AddConfigPath(".")
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
		}
		viper.SetConfigName(defaultConfigFilename)
		viper.SetConfigType("env")
	}

	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err)
	}
}

func initSingleton(ctx context.Context, useArchiveNode bool) error {
	if !useArchiveNode {
		singleton.InitPoolsSDK(
			ctx,
			viper.GetInt64("chain_id"),
			viper.GetString("lotus_private_addr"),
			viper.GetString("lotus_private_token"),
		)

		err := singleton.ConnectLotus(singleton.ChainOptions{
			DialAddr: viper.GetString("lotus_private_addr"),
			Token:    viper.GetString("lotus_private_token"),
		})
		if err != nil {
			return fmt.Errorf("failed to connect to lotus node: %v", err)
		}
	} else {
		singleton.InitPoolsArchiveSDK(
			ctx,
			viper.GetInt64("chain_id"),
			viper.GetString("lotus_archive_addr"),
			viper.GetString("lotus_archive_token"),
		)

		err := singleton.ConnectArchiveLotus(singleton.ChainOptions{
			DialAddr: viper.GetString("lotus_archive_addr"),
			Token:    viper.GetString("lotus_archive_token"),
		})
		if err != nil {
			return fmt.Errorf("failed to connect to lotus archive node: %v", err)
		}
	}
	return nil
}
