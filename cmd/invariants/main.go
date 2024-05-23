package main

import (
	"context"
	"fmt"
	"os"

	"github.com/glifio/invariants/singleton"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "invariants",
		Short: "Checks values from REST API against Lotus node values",
	}
)

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	viper.BindEnv("chain_id")
	viper.BindEnv("events_api")
	viper.BindEnv("lotus_private_token")
	viper.BindEnv("lotus_private_addr")
}

func initConfig() {
	viper.AutomaticEnv()
}

func initSingleton(ctx context.Context) error {
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
	return nil
}
