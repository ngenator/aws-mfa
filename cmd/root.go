// Copyright © 2018 Daniel Ng <dan@ngenator.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/spf13/cobra"
	"github.com/ngenator/aws-mfa/mfa"
)

var Version string

var (
	credentialsFile string
	profile         string
	mfaSerial       string
	duration        time.Duration
	suffix          string
	force           bool
	verbose         bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Version: Version,
	Use:   "aws-mfa <profile>",
	Args: cobra.ExactArgs(1),
	Short: "Refreshes or generates temporary AWS credentials",
	Long: `Refreshes or generates temporary AWS credentials via STS. If you use the '--mfa' flag, the ARN will be
stored in the credentials file so you don't have to pass it every time. If you already have credentials with an
expiration that's an hour out or further, they won't be refreshed unless you use the '--force' flag.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		profile = args[0]
	},
	Run: func(cmd *cobra.Command, args []string) {
		options := mfa.Options{
			CredentialsFileLocation: credentialsFile,
			Profile:                 profile,
			ProfileSuffix:           suffix,
			MFASerial:               mfaSerial,
			Duration:                duration,
			Force:                   force,
			Verbose:                 verbose,
		}

		refresher := mfa.NewRefresher(options)
		refresher.Refresh()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.MarkFlagRequired("profile")
	rootCmd.MarkFlagFilename("credentials")

	rootCmd.PersistentFlags().StringVarP(&credentialsFile, "credentials", "c", external.DefaultSharedCredentialsFilename(), "path to AWS shared credentials file")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "force a refresh even if unexpired credentials exist")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose logging")
	rootCmd.Flags().DurationVarP(&duration, "duration", "d", time.Hour*36, "amount of time the temporary credentials are valid, min: 15m, max: 36h")
	rootCmd.Flags().StringVarP(&suffix, "suffix", "s", "permanent", "suffix to append to profile, used to find permanent credentials. results in <profile>-<suffix>")
	rootCmd.Flags().StringVarP(&mfaSerial, "mfa", "m", "", "arn of your mfa device, e.g. `arn:aws:iam::<account-id>:mfa/<user>` uses one defined in the credentials file if exists and omitted")
}
