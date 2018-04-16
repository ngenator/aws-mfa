package cmd

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/go-ini/ini"
	"github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
)

const (
	// Static Credentials group
	accessKeyIDKey  = `aws_access_key_id`     // group required
	secretAccessKey = `aws_secret_access_key` // group required

	// Optional
	sessionTokenKey = `aws_session_token` // optional

	// Our additional keeys
	mfaSerialKey = `mfa_serial`
	expiresKey   = `expires`
)

var log = logrus.New()

func init() {
	log.Formatter = &prefixed.TextFormatter{
		ForceColors: true,
	}
}

type Options struct {
	CredentialsFileLocation string
	Profile                 string
	ProfileSuffix           string
	Duration                time.Duration
	MFASerial               string
	Force                   bool
	Verbose                 bool
}

func (o Options) GetSourceProfile() string {
	return o.Profile + "-" + o.ProfileSuffix
}

func (o Options) GetMFAToken() (string, error) {
	var v string
	log.Printf("Enter the MFA token code for device %s", o.MFASerial)
	_, err := fmt.Scanln(&v)
	return v, err
}

type Refresher struct {
	log              *logrus.Entry
	CredentialsFile  *ini.File
	PermanentSection *ini.Section
	TemporarySection *ini.Section
	Options          Options
}

func NewRefresher(options Options) Refresher {
	if options.Verbose {
		log.SetLevel(logrus.DebugLevel)
	}

	logger := log.WithField("prefix", "refresher")

	credentialsFile, err := ini.Load(options.CredentialsFileLocation)
	if err != nil {
		logger.WithError(err).Fatalln("Failed to load the credentials file")
	}

	permSection, err := credentialsFile.GetSection(options.GetSourceProfile())
	if err != nil {
		logger.WithError(err).Fatalln("Failed to read permanent credentials section")
	}

	if options.MFASerial == "" && permSection.HasKey(mfaSerialKey) {
		options.MFASerial = permSection.Key(mfaSerialKey).String()
	} else {
		logger.Fatalln("No mfa serial found, please check help for instructions on how to set it")
	}

	logger.WithFields(logrus.Fields{
		"force":             options.Force,
		"permanent-profile": fmt.Sprintf("%s-%s", options.Profile, options.ProfileSuffix),
		"profile":           options.Profile,
		"credentials":       options.CredentialsFileLocation,
		"mfa-serial":        options.MFASerial,
		"duration":          options.Duration.Seconds(),
	}).Debugln("Using the following options")

	tempSection, err := credentialsFile.GetSection(options.Profile)
	if err != nil {
		logger.WithError(err).Debugln("Failed to read temporary credentials section, creating one")
		tempSection = credentialsFile.Section(options.Profile)
	}

	return Refresher{
		log:              logger,
		CredentialsFile:  credentialsFile,
		PermanentSection: permSection,
		TemporarySection: tempSection,
		Options:          options,
	}
}

func (r Refresher) Clear() {
	r.TemporarySection.DeleteKey(accessKeyIDKey)
	r.TemporarySection.DeleteKey(secretAccessKey)
	r.TemporarySection.DeleteKey(sessionTokenKey)
	r.TemporarySection.DeleteKey(expiresKey)

	if err := r.CredentialsFile.SaveTo(r.Options.CredentialsFileLocation); err != nil {
		r.log.WithError(err).Errorln("Failed to clear the temporary credentials")
	}
}

func (r Refresher) Save(credentials *sts.Credentials) {
	r.PermanentSection.Key(mfaSerialKey).SetValue(r.Options.MFASerial)

	r.TemporarySection.Key(accessKeyIDKey).SetValue(aws.StringValue(credentials.AccessKeyId))
	r.TemporarySection.Key(secretAccessKey).SetValue(aws.StringValue(credentials.SecretAccessKey))
	r.TemporarySection.Key(sessionTokenKey).SetValue(aws.StringValue(credentials.SessionToken))
	r.TemporarySection.Key(expiresKey).SetValue(aws.TimeValue(credentials.Expiration).Local().Format(time.RFC3339))

	if err := r.CredentialsFile.SaveTo(r.Options.CredentialsFileLocation); err != nil {
		r.log.WithError(err).Fatalln("Failed to save the temporary credentials")
	}

	r.log.WithFields(logrus.Fields{
		"expires": time.Until(credentials.Expiration.Local()),
		"profile": r.Options.Profile,
	}).Infoln("Successfully refreshed your temporary credentials")
}

func (r Refresher) Refresh() {
	expires := time.Now()
	if r.TemporarySection.HasKey(expiresKey) {
		expires, _ = r.TemporarySection.Key(expiresKey).Time()
	}

	// only refresh if force is set or if the credentials are expired
	r.log.Infoln(expires)
	if r.Options.Force || expires.Before(time.Now().Add(time.Hour)) {
		r.log.WithField("profile", r.Options.Profile).Infoln("Refreshing temporary credentials")

		permConfig, err := external.LoadDefaultAWSConfig(
			external.WithSharedConfigProfile(r.Options.GetSourceProfile()),
			external.WithSharedConfigFiles([]string{r.Options.CredentialsFileLocation}),
		)
		if err != nil {
			r.log.WithError(err).Fatalln("Failed to load your credentials")
		}

		svc := sts.New(permConfig)

		// grab the mfa token if a mfa serial is provided
		token := ""
		if r.Options.MFASerial != "" {
			token, err = r.Options.GetMFAToken()
			if err != nil {
				r.log.WithError(err).Fatalln("Couldn't read your mfa token")
			}
		}

		// build the request to send to STS
		req := svc.GetSessionTokenRequest(&sts.GetSessionTokenInput{
			DurationSeconds: aws.Int64(int64(r.Options.Duration.Seconds())),
			SerialNumber:    aws.String(r.Options.MFASerial),
			TokenCode:       aws.String(token),
		})
		resp, err := req.Send()
		if err != nil {
			r.Clear()
			r.log.WithError(err).Fatalln("Failed to get session token from STS")
		}

		// save the temporary credentials to the credentials file
		r.Save(resp.Credentials)
	} else {
		r.log.Infoln("Already have credentials that expire in", time.Until(expires))
	}
}
