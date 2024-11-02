package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "streakode",
		Short: "Streakode is a developer motivation and insight tool",
		Long:  `Track your coding streaks, get insights, and stay motivated with Streakode`,
	}

	var scanCmd = &cobra.Command{
		Use:   "scan",
		Short: "Scan for Git repositories",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Scanning for Git repositories...")
			// TODO: Implement repository scanning
		},
	}

	var statsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Show your coding stats and streaks",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Showing stats...")
			// TODO: Implement stats display
		},
	}

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(statsCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
