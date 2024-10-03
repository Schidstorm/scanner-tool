package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path"

	"github.com/schidstorm/scanner-tool/pkg/scan"
	"github.com/schidstorm/scanner-tool/pkg/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func main() {
	cmd := &cobra.Command{
		Use:   "scanner-tool",
		Short: "Start the scanner server",
		Run:   helpInterceptor(startServer),
	}

	cmd.Flags().String("config", "", "Path to the configuration file")
	cmd.Flags().Bool("fake", false, "run with fake scanner")

	printConfigCmd := &cobra.Command{
		Use:   "print-config",
		Short: "Print the default configuration",
		Run:   helpInterceptor(printConfig),
	}

	cmd.AddCommand(printConfigCmd)

	err := cmd.Execute()
	if err != nil {
		logrus.WithError(err).Error("Failed to execute command")
	}
}

func helpInterceptor(child func(cmd *cobra.Command, args []string)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		printHelp := false

		for _, arg := range args {
			if arg == "--help" || arg == "-h" {
				printHelp = true
			}
		}

		if printHelp {
			cmd.Help()
			os.Exit(0)
		} else {
			child(cmd, args)
		}
	}
}

func printConfig(cmd *cobra.Command, args []string) {
	var opts server.Options
	fmt.Println("JSON:")
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	enc.Encode(opts)
	fmt.Println("YAML:")
	yaml.NewEncoder(os.Stdout).Encode(opts)
}

func startServer(cmd *cobra.Command, args []string) {
	configPath, _ := cmd.Flags().GetString("config")
	opts, err := parseConfig(configPath)
	if err != nil {
		logrus.WithError(err).Error("Failed to parse config")
		return
	}

	s := server.NewServer(opts)
	if fake, _ := cmd.Flags().GetBool("fake"); fake {
		s.WithScanner(scan.NewFakeScanner())
	}

	s.Start()

	// Wait for a signal to stop the server
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)
	<-signalChannel

	s.Stop()
}

func parseConfig(p string) (server.Options, error) {
	var opts server.Options
	fileContent, err := os.ReadFile(p)
	if err != nil {
		return opts, err
	}

	if path.Ext(p) == ".json" {
		json.Unmarshal(fileContent, &opts)
	} else if path.Ext(p) == ".yaml" || path.Ext(p) == ".yml" {
		yaml.Unmarshal(fileContent, &opts)
	} else {
		return opts, fmt.Errorf("unsupported file format")
	}

	return opts, nil
}
