package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/lostcoords/wariwari-cli/internal/client"
	"github.com/lostcoords/wariwari-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	version = "dev"

	tokenFlag   string
	profileFlag string
	apiURLFlag  string
	groupFlag   string
	jsonOutput  bool
	quiet       bool
	agent       bool
	idsOnly     bool
	countOnly   bool
	markdown    bool
	styled      bool
	verbose     bool
	jqFilter    string
	limit       int

	cfg    *config.Config
	api    *client.Client
	out    io.Writer = os.Stdout
	errOut io.Writer = os.Stderr
)

var rootCmd = &cobra.Command{
	Use:           "wariwari",
	Short:         "WariWari CLI - manage WariWari groups and expenses",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		loadStoredToken := cfg == nil
		if cfg == nil {
			cfg = config.Load()
			config.ResolveProfile(cfg, profileFlag)
		}
		if apiURLFlag != "" {
			cfg.APIURL = apiURLFlag
		}
		if groupFlag != "" {
			cfg.Group = groupFlag
		}
		if tokenFlag != "" {
			cfg.Token = tokenFlag
		} else if loadStoredToken && cfg.Token == "" {
			if token, _ := config.LoadToken(cfg.Profile); token != "" {
				cfg.Token = token
			}
		}
		api = client.New(cfg.APIURL, cfg.Token, version)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.SetVersionTemplate("wariwari {{.Version}}\n")
	rootCmd.PersistentFlags().StringVar(&tokenFlag, "token", "", "API bearer token")
	rootCmd.PersistentFlags().StringVar(&profileFlag, "profile", "", "Named profile to use")
	rootCmd.PersistentFlags().StringVar(&apiURLFlag, "api-url", "", "API base URL")
	rootCmd.PersistentFlags().StringVar(&groupFlag, "group", "", "Default group token")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "JSON envelope output")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Raw JSON data without envelope")
	rootCmd.PersistentFlags().BoolVar(&idsOnly, "ids-only", false, "Print one ID or token per line")
	rootCmd.PersistentFlags().BoolVar(&countOnly, "count", false, "Print count of results")
	rootCmd.PersistentFlags().BoolVar(&markdown, "markdown", false, "Markdown formatted output")
	rootCmd.PersistentFlags().BoolVar(&styled, "styled", false, "Styled terminal output")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Show request/response details")
	rootCmd.PersistentFlags().BoolVar(&agent, "agent", false, "Agent mode")
	rootCmd.PersistentFlags().IntVar(&limit, "limit", 0, "Maximum number of results")
	rootCmd.PersistentFlags().StringVar(&jqFilter, "jq", "", "Apply jq filter to JSON output")
}

func SetVersion(v string) {
	if v != "" {
		version = v
		rootCmd.Version = v
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		printErr(err)
		os.Exit(1)
	}
}

func requireAuth() error {
	if cfg.Token == "" {
		return fmt.Errorf("no API token configured. Run `wariwari auth login TOKEN` or set WARIWARI_TOKEN")
	}
	return nil
}

func requireGroup(group string) (string, error) {
	if group != "" {
		return group, nil
	}
	if cfg.Group != "" {
		return cfg.Group, nil
	}
	return "", fmt.Errorf("no group configured. Pass --group, set WARIWARI_GROUP, or run `wariwari config set-group TOKEN`")
}
