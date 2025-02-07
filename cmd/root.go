/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var version string
var releaseData string
// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "goml",
	Short: "goml is a Go implementation of the mLayer decentralized messaging protocol",
	Long: `mLayer (message layer) is an open, decentralized communication network that enables the creation, 
	transmission and termination of data of all sizes, leveraging modern protocols. mLayer is a comprehensive 
	suite of communication protocols designed to evolve with the ever-advancing realm of cryptography. 
	Given its protocol-centric nature, it is an adaptable and universally integrable tool conceived for the 
	decentralized era. Visit the mLayer [documentation](https://mlayer.gitbook.io/introduction/what-is-mlayer) to learn more
	.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) {
	// 	daemonFunc(cmd, args)
	// },
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "returns the version",
	Long: `mLayer (message layer) is an open, decentralized communication network that enables the creation, 
	transmission and termination of data of all sizes, leveraging modern protocols. mLayer is a comprehensive 
	suite of communication protocols designed to evolve with the ever-advancing realm of cryptography. 
	Given its protocol-centric nature, it is an adaptable and universally integrable tool conceived for the 
	decentralized era. Visit the mLayer [documentation](https://mlayer.gitbook.io/introduction/what-is-mlayer) to learn more
	.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:", version)
		fmt.Println("Release Date:", strings.Replace(releaseData, "_", " ", -1))
	},
}


// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(_version string, _releaseData string) {
	version = _version
	releaseData = _releaseData
	rootCmd.AddCommand(versionCmd)
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// func init() {
// 	// Here you will define your flags and configuration settings.
// 	// Cobra supports persistent flags, which, if defined here,
// 	// will be global for your application.

// 	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.splanch.yaml)")

// 	// Cobra also supports local flags, which will only run
// 	// when this action is called directly.
// 	rootCmd.Flags().StringP("version", "v", "", "Prints out the build version")
	
// }
