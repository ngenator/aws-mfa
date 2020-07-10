module github.com/ruelala/aws-mfa

go 1.14

require (
	github.com/aws/aws-sdk-go-v2 v0.23.0
	github.com/go-ini/ini v1.57.0
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
)

replace github.com/ruelala/aws-mfa/cmd => ./cmd

replace github.com/ruelala/aws-mfa/mfa => ./mfa
