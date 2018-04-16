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
)

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
	Use:   "aws-mfa",
	Short: "Refreshes or generates temporary AWS credentials",
	Long: `Refreshes or generates temporary AWS credentials via STS. If you already have credentials with an
expiration that's an hour out or further, they won't be refreshed unless you use the '--force' flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		options := Options{
			CredentialsFileLocation: credentialsFile,
			Profile:                 profile,
			ProfileSuffix:           suffix,
			MFASerial:               mfaSerial,
			Duration:                duration,
			Force:                   force,
			Verbose:                 verbose,
		}

		refresher := NewRefresher(options)
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

	rootCmd.Flags().StringVarP(&credentialsFile, "credentials", "c", external.DefaultSharedCredentialsFilename(), "Path to the AWS credentials file")
	rootCmd.Flags().StringVarP(&profile, "profile", "p", "", "The profile for which we will generate temporary credentials")
	rootCmd.Flags().DurationVarP(&duration, "duration", "d", time.Hour*36, "Duration for which the temporary credentials are valid. Min: 15m, Max: 36h")
	rootCmd.Flags().StringVarP(&suffix, "suffix", "s", "permanent", "The suffix we will append to the profile where the temporary credentials are stored, ends up in the form <profile>-<suffix>")
	rootCmd.Flags().StringVarP(&mfaSerial, "mfa", "m", "", "The arn of your mfa device, e.g. `arn:aws:iam::<account-id>:mfa/<user>` uses one defined in the credentials file if exists and omitted")
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Force a refresh even if unexpired credentials exist")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
}
