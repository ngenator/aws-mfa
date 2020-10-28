package mfa

import (
  "context"
  "fmt"
  "time"

  "github.com/aws/aws-sdk-go-v2/aws"
  "github.com/aws/aws-sdk-go-v2/config"
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

  // Our additional keys
  mfaSerialKey = `mfa_serial`
  expiresKey   = `expires`
)

var log = logrus.New()

func init() {
  log.Formatter = &prefixed.TextFormatter{
    ForceColors: true,
  }
}

type AWSDebugLogger struct {
  logger *logrus.Entry
}

func (l AWSDebugLogger) Log(args ...interface{}) {
  l.logger.Debugln(args...)
}

func NewAWSDebugLogger(from *logrus.Entry) AWSDebugLogger {
  return AWSDebugLogger{
    logger: from,
  }
}

type ConfigValue struct {
  Profile string
  Section *ini.Section
}

type Config struct {
  Options Options

  Permanent ConfigValue
  Temporary ConfigValue

  CredentialsFile *ini.File
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

func (o Options) Validate() (*Config, error) {
  logger := log.WithField("prefix", "options")

  if o.Verbose {
    log.SetLevel(logrus.DebugLevel)
  }

  logger.Debugln("Validating options")

  logger.WithFields(logrus.Fields{
    "--credentials": o.CredentialsFileLocation,
    "--profile":     o.Profile,
    "--suffix":      o.ProfileSuffix,
    "--duration":    o.Duration,
    "--mfa":         o.MFASerial,
    "--force":       o.Force,
    "--verbose":     o.Verbose,
  }).Debugln("Using the following options")

  permanentProfile := o.Profile + "-" + o.ProfileSuffix

  credentialsFile, err := ini.Load(o.CredentialsFileLocation)
  if err != nil {
    logger.WithError(err).Fatalln("Failed to load the credentials file")
    return nil, err
  }

  perm, err := credentialsFile.GetSection(permanentProfile)
  if err != nil {
    logger.Errorln("Failed to read permanent credentials section")
    return nil, err
  }

  temp, err := credentialsFile.GetSection(o.Profile)
  if err != nil {
    logger.Debugln("Failed to read temporary credentials section, creating one")
    temp = credentialsFile.Section(o.Profile)
  }

  if o.MFASerial == "" && perm.HasKey(mfaSerialKey) {
    o.MFASerial = perm.Key(mfaSerialKey).String()
  }

  return &Config{
    Options: o,

    Permanent: ConfigValue{
      Profile: permanentProfile,
      Section: perm,
    },
    Temporary: ConfigValue{
      Profile: o.Profile,
      Section: temp,
    },
    CredentialsFile: credentialsFile,
  }, nil
}

type Refresher struct {
  log    *logrus.Entry
  Config *Config
}

func NewRefresher(c *Config) (*Refresher, error) {
  return &Refresher{
    log:    log.WithField("prefix", "refresher"),
    Config: c,
  }, nil
}

func (r Refresher) GetMFAToken() (string, error) {
  device := r.Config.Options.MFASerial
  if device == "" {
    r.log.Printf("No MFA serial found, please enter one")
    if _, err := fmt.Scanln(&device); err != nil {
      r.log.Errorln("Can't continue without a MFA serial")
      return "", err
    }
  }

  r.Config.Options.MFASerial = device

  var token string
  r.log.Printf("Enter the MFA token code for device %s", device)
  _, err := fmt.Scanln(&token)
  return token, err
}

func (r Refresher) Clear(removeMfa bool) error {
  if removeMfa {
    r.log.Infoln("Clearing mfa device from permanent section")
    r.Config.Permanent.Section.DeleteKey(mfaSerialKey)
  }

  r.log.Debugln("Clearing credentials from temporary section")

  r.Config.Temporary.Section.DeleteKey(accessKeyIDKey)
  r.Config.Temporary.Section.DeleteKey(secretAccessKey)
  r.Config.Temporary.Section.DeleteKey(sessionTokenKey)
  r.Config.Temporary.Section.DeleteKey(expiresKey)

  if err := r.Config.CredentialsFile.SaveTo(r.Config.Options.CredentialsFileLocation); err != nil {
    r.log.WithError(err).Errorln("Failed to clear the temporary credentials")
    return err
  }

  return nil
}

func (r Refresher) Save(credentials *sts.Credentials) error {
  if r.Config.Options.MFASerial != "" {
    oldSerial := r.Config.Permanent.Section.Key(mfaSerialKey).String()
    newSerial := r.Config.Options.MFASerial
    if oldSerial != newSerial  {
      r.log.WithFields(logrus.Fields{"old": oldSerial, "new": newSerial}).Infoln("Updating saved MFA serial")
    } else {
      r.log.Infoln("Saving MFA serial to permanent section")
    }
    r.Config.Permanent.Section.Key(mfaSerialKey).SetValue(newSerial)
  }

  r.log.Infoln("Saving credentials to temporary section")

  r.Config.Temporary.Section.Key(accessKeyIDKey).SetValue(aws.StringValue(credentials.AccessKeyId))
  r.Config.Temporary.Section.Key(secretAccessKey).SetValue(aws.StringValue(credentials.SecretAccessKey))
  r.Config.Temporary.Section.Key(sessionTokenKey).SetValue(aws.StringValue(credentials.SessionToken))
  r.Config.Temporary.Section.Key(expiresKey).SetValue(aws.TimeValue(credentials.Expiration).Local().Format(time.RFC3339))

  if err := r.Config.CredentialsFile.SaveTo(r.Config.Options.CredentialsFileLocation); err != nil {
    r.log.Errorln("Failed to save the temporary credentials")
    return err
  }

  return nil
}

func (r Refresher) Refresh() error {
  expires := time.Now()
  if r.Config.Temporary.Section.HasKey(expiresKey) {
    expires, _ = r.Config.Temporary.Section.Key(expiresKey).Time()
  }

  // only refresh if force is set or if the credentials are expired
  if r.Config.Options.Force || expires.Before(time.Now().Add(time.Hour)) {
    r.log.WithField("profile", r.Config.Options.Profile).Infoln("Refreshing temporary credentials")

    awsConfig, err := config.LoadDefaultConfig(
      config.WithSharedConfigProfile(r.Config.Permanent.Profile),
      config.WithSharedConfigFiles([]string{r.Config.Options.CredentialsFileLocation}),
    )

    svc := sts.New(awsConfig)

    // build the request to send to STS
    input := &sts.GetSessionTokenInput{
      DurationSeconds: aws.Int64(int64(r.Config.Options.Duration.Seconds())),
    }

    if r.Config.Options.MFASerial != "" {
      var token string
      token, err = r.GetMFAToken()
      if err != nil {
        r.log.WithError(err).Fatalln("Couldn't read your MFA token")
      }
      input.SerialNumber = aws.String(r.Config.Options.MFASerial)
      input.TokenCode = aws.String(token)
    } else {
      r.log.Warnln("No MFA Serial provided, your temporary credentials may not work as expected")
      r.log.Infoln("Use --mfa to provide an MFA device")
    }

    // send the request to STS
    req := svc.GetSessionTokenRequest(input)
    resp, err := req.Send(context.Background())
    if err != nil {
      r.log.WithError(err).Errorln("Failed to get session token from STS")
      r.Clear(false)
      return err
    }

    // save the temporary credentials to the credentials file
    if err := r.Save(resp.Credentials); err != nil {
      return err
    }

    r.log.WithFields(logrus.Fields{
      "expires": time.Until(resp.Credentials.Expiration.Local()),
      "profile": r.Config.Options.Profile,
    }).Println("Successfully refreshed your temporary credentials")
  } else {
    r.log.Println("Already have credentials that expire in", time.Until(expires))
    r.log.Infoln("Use --force to update anyways")
  }

  return nil
}
