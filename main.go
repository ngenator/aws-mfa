package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/go-ini/ini"
)

func get_config(profile string) aws.Config {
	cfg, _ := config.LoadDefaultConfig(
		config.WithSharedConfigProfile(profile),
		config.WithSharedConfigFiles(
			[]string{
				config.DefaultSharedConfigFilename(),
				config.DefaultSharedCredentialsFilename(),
			}),
	)
	return cfg
}

func ini_section_exists(profile string) bool {
	cfg, _ := ini.Load(config.DefaultSharedCredentialsFilename())
	section := cfg.Section(profile)
	if len(section.Keys()) == 0 {
		return false
	} else {
		return true
	}
}

func get_ini_val(profile, key string) *ini.Key {
	cfg, _ := ini.Load(config.DefaultSharedCredentialsFilename())
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

func get_mfa_token() string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter token: ")
	scanner.Scan()
	return scanner.Text()
}

func get_session_creds(client *sts.Client, mfa_serial, mfa_token string) *types.Credentials {
	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int32(int32(129600)),
		SerialNumber:    aws.String(mfa_serial),
		TokenCode:       aws.String(mfa_token),
	}
	resp, _ := client.GetSessionToken(context.Background(), input)
	return resp.Credentials
}

func write_creds(profile string, creds *types.Credentials) {
	cfg, _ := ini.Load(config.DefaultSharedCredentialsFilename())
	cfg.Section(profile).Key("aws_access_key_id").SetValue(aws.ToString(creds.AccessKeyId))
	cfg.Section(profile).Key("aws_secret_access_key").SetValue(aws.ToString(creds.SecretAccessKey))
	cfg.Section(profile).Key("aws_session_token").SetValue(aws.ToString(creds.SessionToken))
	cfg.Section(profile).Key("expires").SetValue(aws.ToTime(creds.Expiration).Local().Format(time.RFC3339))
	cfg.SaveTo(config.DefaultSharedCredentialsFilename())
}

func main() {
	// get permanent credentials info
	permanent := get_config("rue-ops-permanent")

	// check for existing mfa credentials and load them
	exists := ini_section_exists("rue-ops")
	if exists {
		expire_time, _ := get_ini_val("rue-ops", "expires").Time()
		if time.Now().Before(expire_time) {
			fmt.Println("creds havent expired yet, use -f to force renewal")
			os.Exit(0)
		}
	}

	client := sts_client(permanent)
	mfa_token := get_mfa_token()
	mfa_serial := get_ini_val("rue-ops-permanent", "mfa_serial").String()
	credentials := get_session_creds(client, mfa_serial, mfa_token)
	write_creds("rue-ops", credentials)
}
