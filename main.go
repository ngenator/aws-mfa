// Copyright © 2018 Daniel Ng <dan@RueLaLa.com>
// Copyright © 2020 Nick Silverman <nckslvrmn@gmail.com>
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
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/go-ini/ini"
)

func panic(err error) {
	if err != nil {
		log.Printf("ERROR: %s\n", err)
	}
}

func get_config(profile string) aws.Config {
	cfg, err := config.LoadDefaultConfig(
		config.WithSharedConfigProfile(profile),
		config.WithSharedConfigFiles(
			[]string{
				config.DefaultSharedConfigFilename(),
				config.DefaultSharedCredentialsFilename(),
			}),
	)
	panic(err)
	return cfg
}

func ini_section_exists(profile string) bool {
	cfg, err := ini.Load(config.DefaultSharedCredentialsFilename())
	panic(err)
	section := cfg.Section(profile)
	if len(section.Keys()) == 0 {
		return false
	} else {
		return true
	}
}

func get_ini_val(profile, key string) *ini.Key {
	cfg, err := ini.Load(config.DefaultSharedCredentialsFilename())
	panic(err)
	val := cfg.Section(profile).Key(key)
	return val
}

func sts_client(permanent aws.Config) *sts.Client {
	cli_opt := sts.Options{
		Credentials: permanent.Credentials,
		Region:      "us-east-1",
	}
	client := sts.New(cli_opt)
	return client
}

func get_mfa_token(mfa_serial string) string {
	scanner := bufio.NewScanner(os.Stdin)
	log.Printf("INFO: Enter the MFA token code for device %s below\n", mfa_serial)
	scanner.Scan()
	return scanner.Text()
}

func get_session_creds(client *sts.Client, mfa_serial, mfa_token string) *types.Credentials {
	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int32(int32(129600)),
		SerialNumber:    aws.String(mfa_serial),
		TokenCode:       aws.String(mfa_token),
	}
	resp, err := client.GetSessionToken(context.Background(), input)
	panic(err)
	return resp.Credentials
}

func write_creds(profile string, creds *types.Credentials) {
	cfg, err := ini.Load(config.DefaultSharedCredentialsFilename())
	panic(err)
	cfg.Section(profile).Key("aws_access_key_id").SetValue(aws.ToString(creds.AccessKeyId))
	cfg.Section(profile).Key("aws_secret_access_key").SetValue(aws.ToString(creds.SecretAccessKey))
	cfg.Section(profile).Key("aws_session_token").SetValue(aws.ToString(creds.SessionToken))
	cfg.Section(profile).Key("expires").SetValue(aws.ToTime(creds.Expiration).Local().Format(time.RFC3339))
	cfg.SaveTo(config.DefaultSharedCredentialsFilename())
}

type Args struct {
	profile string
	force   bool
	suffix  string
}

func parse_args() Args {
	a := Args{}
	flag.StringVar(&a.profile, "profile", "default", "profile to create MFA creds with")
	flag.StringVar(&a.profile, "p", "default", "profile to create MFA creds with")
	flag.BoolVar(&a.force, "force", false, "force MFA recreation regardless of existing tokens")
	flag.BoolVar(&a.force, "f", false, "force MFA recreation regardless of existing tokens")
	flag.StringVar(&a.suffix, "suffix", "permanent", "suffix to match to find static credentials file")
	flag.StringVar(&a.suffix, "s", "permanent", "suffix to match to find static credentials file")
	flag.Parse()
	return a
}

func main() {
	args := parse_args()
	log.SetFlags(log.Ltime)
	perm_profile := fmt.Sprintf("%s-%s", args.profile, args.suffix)

	// get permanent credentials info
	permanent := get_config(perm_profile)
	if !ini_section_exists(perm_profile) {
		log.Printf("ERROR: couldnt find %s profile in aws credentials file\n", perm_profile)
		os.Exit(1)
	}

	// check for existing mfa credentials and load them
	if ini_section_exists(args.profile) {
		expire_time, err := get_ini_val(args.profile, "expires").Time()
		panic(err)
		if !args.force && time.Now().Before(expire_time) {
			log.Println("INFO: creds havent expired yet, use -f/-force to force renewal")
			os.Exit(0)
		}
	}
	log.Printf("INFO: Refreshing temporary credentials for %s profile", args.profile)

	client := sts_client(permanent)
	mfa_serial := get_ini_val(perm_profile, "mfa_serial").String()
	mfa_token := get_mfa_token(mfa_serial)
	credentials := get_session_creds(client, mfa_serial, mfa_token)
	write_creds(args.profile, credentials)
	log.Printf("INFO: Successfully refreshed temporary credentials for %s profile (expires: %s)", args.profile, aws.ToTime(credentials.Expiration).Local().Format(time.RFC3339))
}
