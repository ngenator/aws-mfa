# aws-mfa [![Build Status](https://travis-ci.org/ngenator/aws-mfa.svg?branch=develop)](https://travis-ci.org/ngenator/aws-mfa)

Highly recommend using [aws-vault](https://github.com/99designs/aws-vault) instead. You can setup your `~/.aws/config` in a way that mimics the functionality here, while keeping your credentials in a secure backend. Use [`credential_process`](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html) which is [supported by aws-vault](https://github.com/99designs/aws-vault/blob/master/USAGE.md#using-credential_process) to provide credentials to profiles with a role and you're good to go.

Generates or refreshes temporary aws credentials via STS and stores them to support tools that don't behave nicely when mfa is required.
To do this, we have the idea of "permanent" credentials and temporary credentials. To support existing scripts/tooling, 
the tool looks for a permanent profile using a suffix rather than generating a temporary profile with one.

## Features

  * Stores generated credentials in your credentials file for reuse
  * Expiration is stored in the credentials file to prevent unnecessary refreshes (can be overridden with `--force`)
  * Stores your mfa serial in the credentials file
  * Customizable suffix for the "permanent" credentials
  * Customizable duration (within the limits of STS)
  

## Install

Head over to [releases](https://github.com/ngenator/aws-mfa/releases) and download the latest version for your OS/Architecture, and place the extracted binary in your PATH.

## Usage 
```
$ ./aws-mfa -h
Refreshes or generates temporary AWS credentials via STS. If you use the '--mfa' flag, the ARN will be
stored in the credentials file so you don't have to pass it every time. If you already have credentials with an
expiration that's an hour out or further, they won't be refreshed unless you use the '--force' flag.

Usage:
  aws-mfa [flags]

Flags:
  -c, --credentials string                         path to AWS shared credentials file (default "/Users/dng/.aws/credentials")
  -d, --duration duration                          amount of time the temporary credentials are valid, min: 15m, max: 36h (default 36h0m0s)
  -f, --force                                      force a refresh even if unexpired credentials exist
  -h, --help                                       help for aws-mfa
  -m, --mfa arn:aws:iam::<account-id>:mfa/<user>   arn of your mfa device, e.g. arn:aws:iam::<account-id>:mfa/<user> uses one defined in the credentials file if exists and omitted
  -p, --profile string                             profile that will contain the temporary credentials within the AWS shared credentials file (default "default")
  -s, --suffix string                              suffix to append to profile, used to find permanent credentials. results in <profile>-<suffix> (default "permanent")
      --verbose                                    enable verbose logging
      --version                                    version for aws-mfa
```

## Example
Basic example with an `mfa_serial` defined in the credentials file and a `default-permanent` section

```
# ~/.aws/credentials

[default-permanent]
aws_access_key_id     = <YOUR_ACCESS_KEY_ID>
aws_secret_access_key = <YOUR_SECRET_ACCESS_KEY>
mfa_serial            = arn:aws:iam::<ACCOUNT_ID>:mfa/<DEVICE>
```

Run `aws-mfa` and follow the prompt to provide your mfa token. After refreshing the tokens, your credentials file will contain a new `default` section.

```
# ~/.aws/credentials
[default-permanent]
aws_access_key_id     = <YOUR_ACCESS_KEY_ID>
aws_secret_access_key = <YOUR_SECRET_ACCESS_KEY>
mfa_serial            = arn:aws:iam::<ACCOUNT_ID>:mfa/<DEVICE>

[default]
aws_access_key_id     = <TEMPORARY_ACCESS_KEY_ID>
aws_secret_access_key = <TEMPORARY_SECRET_ACCESS_KEY>
aws_session_token     = <SESSION_TOKEN>
expires               = 2018-05-12T03:18:07-04:00
```

### Profiles

If you don't provide a profile with the `--profile` flag, it will use the value `default` and look for a `default-permanent` profile to use.

```
$ ./aws-mfa --profile <my-other-profile>
```

## License
The MIT License (MIT)

Copyright © 2018 Daniel Ng <dan@ngenator.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
