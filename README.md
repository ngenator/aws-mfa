# aws-mfa

Generates or refreshes temporary aws credentials via STS and stores them to support tools that don't behave nicely when mfa is required.
To do this, we have the idea of "permanent" credentials and temporary credentials. To support existing scripts/tooling, 
the tool looks for a permanent profile using a suffix rather than generating a temporary profile with one.

## Features

  * Stores credentials in your credentials file for reuse
  * Stores your mfa serial in the credentials file
  * Aware of expiration and doesn't refresh if you already have unexpired credentials
  * Customizable suffix for the "permanent" credentials
  * Expiration is stored in the credentials file to prevent unnecessary refreshes (can be overridden with `--force`)

## Install

Head over to [releases](https://github.com/ngenator/aws-mfa/releases) and download the latest version for your OS/Architecture, and place the extracted binary in your PATH.

## Usage 
```
$ ./aws-mfa -h
Refreshes or generates temporary AWS credentials via STS. If you use the '--mfa' flag, the ARN will be
stored in the credentials file so you don't have to pass it every time. If you already have credentials with an
expiration that's an hour out or further, they won't be refreshed unless you use the '--force' flag.

Usage:
  aws-mfa <profile> [flags]

Flags:
  -c, --credentials string                         path to AWS shared credentials file (default "/Users/dng/.aws/credentials")
  -d, --duration duration                          amount of time the temporary credentials are valid, min: 15m, max: 36h (default 36h0m0s)
  -f, --force                                      force a refresh even if unexpired credentials exist
  -h, --help                                       help for aws-mfa
  -m, --mfa arn:aws:iam::<account-id>:mfa/<user>   arn of your mfa device, e.g. arn:aws:iam::<account-id>:mfa/<user> uses one defined in the credentials file if exists and omitted
  -s, --suffix string                              suffix to append to profile, used to find permanent credentials. results in <profile>-<suffix> (default "permanent")
      --verbose                                    enable verbose logging
      --version                                    version for aws-mfa

```

## Example
Basic example with an `mfa_serial` defined in the credentials file


```
# ~/.aws/credentials

[default-permanent]
aws_access_key_id     = <YOUR_ACCESS_KEY_ID>
aws_secret_access_key = <YOUR_SECRET_ACCESS_KEY>
mfa_serial            = arn:aws:iam::<ACCOUNT_ID>:mfa/<DEVICE>
```

Run `aws-mfa default` and follow the prompt to provide your mfa token

```
# ~/.aws/credentials
...
[default]
aws_access_key_id     = <TEMPORARY_ACCESS_KEY_ID>
aws_secret_access_key = <TEMPORARY_SECRET_ACCESS_KEY>
aws_session_token     = <SESSION_TOKEN>
expires               = 2018-05-12T03:18:07-04:00
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