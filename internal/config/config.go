// Package config reads every knob the process has from the environment. There
// is no config file: a deployment should be readable from its manifest alone.
package config

import (
	"fmt"
	"github.com/damonleelcx/ai-social-network/internal/billing"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr         string
	DatabaseURL  string
	PublicOrigin string // absolute origin every emailed link is built against
	SessionTTL   time.Duration
	CookieSecure bool

	LLMBaseURL   string
	// Pictures for plans and trips (internal/scenes). ImageURL defaults to the LLM host's
	// DashScope multimodal generation; "off" turns pictures off. ImageKey defaults to
	// LLMAPIKey; set it when the pictures come from a different provider than the chat.
	ImageModel, ImageURL, ImageKey string
	SceneDailyCap        int
	LLMAPIKey    string
	LLMModel     string // reasoning, drafting, planning
	LLMFastModel string // routing, judging, summarising

	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string
	SMTPReplyTo string

	// AdminEmails are operators (moderation), comma separated.
	// An address counts only once its owner has verified it.
	AdminEmails []string

	WorkerCount   int
	LeaseDuration time.Duration

	// Turn workers answer people talking to Suki: short jobs, short lease.
	TurnWorkers int
	TurnLease   time.Duration

	// Spending ceilings in tokens. Per-goal limits live on the goal; these
	// bound a whole account and the whole deployment per UTC day.
	AccountDailyTokens    int
	DeploymentDailyTokens int

	// MongoDB holds every chat message.
	MongoURL string
	MongoDB  string

	// S3 holds photo bytes. Empty bucket with PhotoDir set stores on disk (development only).
	S3Bucket   string
	S3Region   string
	S3Endpoint string // MinIO or similar in development
	PhotoDir   string

	// Places (city lookup, venue search). OpenStreetMap by default.
	NominatimURL string
	OverpassURL  string
	PhotonURL    string
	// PhotoAutoApprove skips vision moderation; development only, never set in production.
	PhotoAutoApprove bool
	// Suki's voice (Fish Audio). No key or voice id means no voice: the speech
	// endpoint reports disabled and the app hides its controls.
	TTSAPIKey     string
	TTSVoiceID    string
	TTSModel      string
	TTSDailyChars int
	// TTSProvider is "fish" (default) or "cosyvoice" (Alibaba DashScope, for the China copy).
	TTSProvider string
	// Stripe. No keys means credits can't be bought, so messages to Suki stay free.
	StripeSecretKey     string
	StripeWebhookSecret string
	// WeChat Pay for the mini program. All of it, or none of it.
	WeChat billing.WeChatConfig
	// WeChat's virtual payment for credits in the mini program. All of it, or none of it.
	XPay billing.XPayConfig
	// Google Play Billing: the Android package, and a service account's JSON key.
	// GoogleClientID is the OAuth web client id for sign-in. Empty turns it off.
	GoogleClientID           string
	GooglePlayPackage        string
	GooglePlayServiceAccount string
	// ExpoPushURL is where push notifications go; empty disables push.
	ExpoPushURL string
	// ICPBeian and ICPLicence are the mainland China filing number and telecom licence
	// shown in the home page footer. Set only on the China copy.
	ICPBeian   string
	ICPLicence string
	// Region is this copy ("us" or "cn"); PeerURL and PeerSecret link it to the other one,
	// so an account works from both. Empty PeerURL runs this copy alone.
	Region     string
	PeerURL    string
	PeerSecret string

	// DateMinLead is the soonest a date can be set; DebriefDelay is how long after
	// its start Suki asks how it went. Both shorten for demos and tests.
	DateMinLead  time.Duration
	DebriefDelay time.Duration

	// Rewards are credits earned, not bought: an invited friend going live pays both sides
	// (SYN_INVITE_CREDITS_INVITER, SYN_INVITE_CREDITS_INVITEE), at most SYN_INVITE_DAILY_CAP
	// friends per China day and SYN_INVITE_TOTAL_CAP ever per inviter; a daily check-in pays
	// SYN_CHECKIN_CREDITS. Zero turns a reward off.
	Rewards billing.Rewards
}

func Load() (Config, error) {
	c := Config{
		Addr:                     env("SYN_ADDR", ":8080"),
		DatabaseURL:              os.Getenv("SYN_DATABASE_URL"),
		PublicOrigin:             strings.TrimRight(os.Getenv("SYN_PUBLIC_ORIGIN"), "/"),
		SessionTTL:               envDur("SYN_SESSION_TTL", 30*24*time.Hour),
		CookieSecure:             envBool("SYN_COOKIE_SECURE", true),
		LLMBaseURL:               env("SYN_LLM_BASE_URL", "https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1"),
		LLMAPIKey:                os.Getenv("SYN_LLM_API_KEY"),
		ImageModel:               env("SYN_IMAGE_MODEL", "qwen-image-3.0-pro"),
		ImageURL:                 os.Getenv("SYN_IMAGE_URL"),
		ImageKey:                 os.Getenv("SYN_IMAGE_KEY"),
		SceneDailyCap:            envInt("SYN_SCENE_DAILY_CAP", 100),
		StripeSecretKey:          os.Getenv("SYN_STRIPE_SECRET_KEY"),
		StripeWebhookSecret:      os.Getenv("SYN_STRIPE_WEBHOOK_SECRET"),
		GooglePlayPackage:        env("SYN_GOOGLE_PLAY_PACKAGE", "space.herosagent.synagent"),
		GoogleClientID:           os.Getenv("SYN_GOOGLE_CLIENT_ID"),
		GooglePlayServiceAccount: os.Getenv("SYN_GOOGLE_PLAY_SERVICE_ACCOUNT"),
		WeChat: billing.WeChatConfig{
			AppID:         os.Getenv("SYN_WECHAT_APP_ID"),
			AppSecret:     os.Getenv("SYN_WECHAT_APP_SECRET"),
			MchID:         os.Getenv("SYN_WECHAT_MCH_ID"),
			MchSerial:     os.Getenv("SYN_WECHAT_MCH_SERIAL"),
			APIv3Key:      os.Getenv("SYN_WECHAT_API_V3_KEY"),
			PrivateKeyPEM: os.Getenv("SYN_WECHAT_MCH_PRIVATE_KEY"),
			PublicKeyID:   os.Getenv("SYN_WECHAT_PAY_PUBLIC_KEY_ID"),
			PublicKeyPEM:  os.Getenv("SYN_WECHAT_PAY_PUBLIC_KEY"),
		},
		XPay: billing.XPayConfig{
			OfferID:    os.Getenv("SYN_WECHAT_XPAY_OFFER_ID"),
			AppKey:     os.Getenv("SYN_WECHAT_XPAY_APP_KEY"),
			Env:        os.Getenv("SYN_WECHAT_XPAY_ENV"),
			PushToken:  os.Getenv("SYN_WECHAT_PUSH_TOKEN"),
			PushAESKey: os.Getenv("SYN_WECHAT_PUSH_AES_KEY"),
		},
		LLMModel:              env("SYN_LLM_MODEL", "qwen3.8-max"),
		LLMFastModel:          env("SYN_LLM_FAST_MODEL", "qwen3.8-flash"),
		SMTPHost:              os.Getenv("SYN_SMTP_HOST"),
		SMTPPort:              envInt("SYN_SMTP_PORT", 587),
		SMTPUser:              os.Getenv("SYN_SMTP_USERNAME"),
		SMTPPass:              os.Getenv("SYN_SMTP_PASSWORD"),
		SMTPFrom:              os.Getenv("SYN_SMTP_FROM"),
		SMTPReplyTo:           os.Getenv("SYN_SMTP_REPLY_TO"),
		WorkerCount:           envInt("SYN_WORKERS", 2),
		LeaseDuration:         envDur("SYN_LEASE", 90*time.Second),
		TurnWorkers:           envInt("SYN_TURN_WORKERS", 4),
		TurnLease:             envDur("SYN_TURN_LEASE", 60*time.Second),
		AccountDailyTokens:    envInt("SYN_ACCOUNT_DAILY_TOKENS", 5_000_000),
		DeploymentDailyTokens: envInt("SYN_DEPLOYMENT_DAILY_TOKENS", 40_000_000),
		MongoURL:              os.Getenv("SYN_MONGO_URL"),
		MongoDB:               env("SYN_MONGO_DB", "synagent"),
		S3Bucket:              os.Getenv("SYN_S3_BUCKET"),
		S3Region:              env("SYN_S3_REGION", "us-east-1"),
		S3Endpoint:            os.Getenv("SYN_S3_ENDPOINT"),
		PhotoDir:              os.Getenv("SYN_PHOTO_DIR"),
		NominatimURL:          env("SYN_NOMINATIM_URL", "https://nominatim.openstreetmap.org"),
		OverpassURL:           env("SYN_OVERPASS_URL", "https://overpass-api.de/api/interpreter,https://maps.mail.ru/osm/tools/overpass/api/interpreter,https://overpass.private.coffee/api/interpreter"),
		TTSAPIKey:             os.Getenv("SYN_TTS_API_KEY"),
		TTSVoiceID:            os.Getenv("SYN_TTS_VOICE_ID"),
		TTSModel:              env("SYN_TTS_MODEL", "s2.1-pro-free"),
		TTSDailyChars:         envInt("SYN_TTS_DAILY_CHARS", 20000),
		TTSProvider:           env("SYN_TTS_PROVIDER", "fish"),
		PhotonURL:             env("SYN_PHOTON_URL", "https://photon.komoot.io"),
		PhotoAutoApprove:      envBool("SYN_PHOTO_AUTO_APPROVE", false),
		ExpoPushURL:           env("SYN_EXPO_PUSH_URL", "https://exp.host/--/api/v2/push/send"),
		ICPBeian:              os.Getenv("SYN_ICP_BEIAN"),
		ICPLicence:            os.Getenv("SYN_ICP_LICENCE"),
		Region:                env("SYN_REGION", "us"),
		PeerURL:               os.Getenv("SYN_PEER_URL"),
		PeerSecret:            os.Getenv("SYN_PEER_SECRET"),
		DateMinLead:           envDur("SYN_DATE_MIN_LEAD", time.Hour),
		DebriefDelay:          envDur("SYN_DEBRIEF_DELAY", 14*time.Hour),
	}
	d := billing.DefaultRewards()
	c.Rewards = billing.Rewards{
		InviterCredits: envNonNeg("SYN_INVITE_CREDITS_INVITER", d.InviterCredits),
		InviteeCredits: envNonNeg("SYN_INVITE_CREDITS_INVITEE", d.InviteeCredits),
		InviteDailyCap: envNonNeg("SYN_INVITE_DAILY_CAP", d.InviteDailyCap),
		InviteTotalCap: envNonNeg("SYN_INVITE_TOTAL_CAP", d.InviteTotalCap),
		CheckInCredits: envNonNeg("SYN_CHECKIN_CREDITS", d.CheckInCredits),
	}
	for _, e := range strings.Split(os.Getenv("SYN_ADMIN_EMAILS"), ",") {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			c.AdminEmails = append(c.AdminEmails, e)
		}
	}
	// The relay only lets a login send as its own address (spoof protection),
	// so the login IS the From address unless one is set explicitly.
	if c.SMTPFrom == "" {
		c.SMTPFrom = c.SMTPUser
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("SYN_DATABASE_URL is required")
	}
	if c.MongoURL == "" {
		return c, fmt.Errorf("SYN_MONGO_URL is required")
	}
	// "off" turns push off (the China copy, where Expo isn't used).
	if c.ExpoPushURL == "off" {
		c.ExpoPushURL = ""
	}
	// "off" skips Nominatim, which mainland China can't reach; cities then come from Photon.
	if c.NominatimURL == "off" {
		c.NominatimURL = ""
	}
	if c.S3Bucket == "" && c.PhotoDir == "" {
		return c, fmt.Errorf("SYN_S3_BUCKET is required")
	}
	return c, nil
}

// MailEnabled is false unless every piece a real message needs is present. A
// half-configured relay would fail per message; this fails once, at startup.
func (c Config) MailEnabled() bool {
	return c.SMTPHost != "" && c.SMTPUser != "" && c.SMTPPass != "" && c.SMTPFrom != "" && c.PublicOrigin != ""
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envInt(k string, d int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return d
}

// envNonNeg is envInt that ignores a negative value.
func envNonNeg(k string, d int) int {
	if v := envInt(k, d); v >= 0 {
		return v
	}
	return d
}

func envBool(k string, d bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(k)); err == nil {
		return v
	}
	return d
}

func envDur(k string, d time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(k)); err == nil {
		return v
	}
	return d
}
