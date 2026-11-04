// SynAgent: one binary, several modes, so a deployment can scale the web tier and
// the workers independently.
//
//	synagent serve    web + workers in one process (development)
//	synagent web      the API
//	synagent worker   mission workers, turn workers, scheduler
//	synagent migrate  apply database migrations and exit
//	synagent mailcheck  prove the mail relay accepts our login without sending
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/agent"
	"github.com/damonleelcx/ai-social-network/internal/auth"
	"github.com/damonleelcx/ai-social-network/internal/billing"
	"github.com/damonleelcx/ai-social-network/internal/chatstore"
	"github.com/damonleelcx/ai-social-network/internal/config"
	"github.com/damonleelcx/ai-social-network/internal/dating"
	"github.com/damonleelcx/ai-social-network/internal/db"
	"github.com/damonleelcx/ai-social-network/internal/engine"
	"github.com/damonleelcx/ai-social-network/internal/httpapi"
	"github.com/damonleelcx/ai-social-network/internal/llm"
	"github.com/damonleelcx/ai-social-network/internal/mail"
	"github.com/damonleelcx/ai-social-network/internal/notify"
	"github.com/damonleelcx/ai-social-network/internal/photostore"
	"github.com/damonleelcx/ai-social-network/internal/places"
	"github.com/damonleelcx/ai-social-network/internal/tts"
	"github.com/damonleelcx/ai-social-network/internal/workflows"
)

func main() {
	mode := "serve"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(mode); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(mode string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	var mailer mail.Mailer = mail.Log{}
	if cfg.MailEnabled() {
		mailer = &mail.SMTP{Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser, Pass: cfg.SMTPPass, From: cfg.SMTPFrom, ReplyTo: cfg.SMTPReplyTo}
	} else {
		slog.Warn("MAIL_DISABLED: emails are logged, not sent")
	}
	if mode == "mailcheck" {
		s, ok := mailer.(*mail.SMTP)
		if !ok {
			return errors.New("mail relay is not configured")
		}
		return s.Check(ctx)
	}
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	if mode == "migrate" {
		slog.Info("migrations applied")
		return nil
	}
	chats, err := chatstore.Open(ctx, cfg.MongoURL, cfg.MongoDB)
	if err != nil {
		return err
	}
	defer chats.Close(context.Background())
	var photos photostore.Store
	if cfg.S3Bucket != "" {
		if photos, err = photostore.NewS3(ctx, cfg.S3Bucket, cfg.S3Region, cfg.S3Endpoint); err != nil {
			return err
		}
	} else {
		slog.Warn("PHOTOS_ON_DISK: development only", "dir", cfg.PhotoDir)
		photos = &photostore.Disk{Dir: cfg.PhotoDir}
	}
	if err := photos.Check(ctx); err != nil {
		return fmt.Errorf("photo store: %w", err)
	}

	store := &engine.Store{Pool: pool, Conversations: chats}
	client := llm.New(cfg.LLMBaseURL, cfg.LLMAPIKey)
	client.Fallback = map[string]string{cfg.LLMModel: cfg.LLMFastModel}
	model := &engine.Model{Client: client, Store: store, AccountDailyTokens: cfg.AccountDailyTokens, DeploymentDailyTokens: cfg.DeploymentDailyTokens}
	plc := &places.Client{Pool: pool, NominatimURL: cfg.NominatimURL, OverpassURL: cfg.OverpassURL, PhotonURL: cfg.PhotonURL}
	svc := &dating.Service{Pool: pool, Chats: chats, DateMinLead: cfg.DateMinLead}
	bill := &billing.Service{Pool: pool, StripeKey: cfg.StripeSecretKey, WebhookSecret: cfg.StripeWebhookSecret, PublicOrigin: cfg.PublicOrigin}
	if wc, err := billing.NewWeChat(cfg.WeChat); err != nil {
		// Half-configured payments stay off, and shouldn't take the server down.
		slog.Error("wechat pay is off", "err", err)
	} else {
		bill.WeChat = wc
	}
	workflows.Init(&workflows.Deps{Billing: bill, Pool: pool, Store: store, Model: model, Chats: chats, Photos: photos, Places: plc, Dating: svc,
		FastModel: cfg.LLMFastModel, PhotoAutoApprove: cfg.PhotoAutoApprove, DebriefDelay: cfg.DebriefDelay})

	var wg sync.WaitGroup
	if mode == "serve" || mode == "worker" {
		startWorkers(ctx, &wg, cfg, store, model, chats, svc, bill, mailer)
	}
	if mode == "serve" || mode == "web" {
		hub := httpapi.NewHub()
		a := &auth.Service{Pool: pool, Mailer: mailer, PublicOrigin: cfg.PublicOrigin, SessionTTL: cfg.SessionTTL, AdminEmails: cfg.AdminEmails}
		srv := &httpapi.Server{Pool: pool, Auth: a, Chats: chats, Photos: photos, Places: plc, Dating: svc, Hub: hub, Billing: bill}
		a.BeforeDelete = srv.BeforeDelete
		srv.SpeechDailyChars = cfg.TTSDailyChars
		if cfg.TTSAPIKey != "" && cfg.TTSVoiceID != "" {
			voice, err := tts.NewFish("", cfg.TTSAPIKey, cfg.TTSVoiceID, cfg.TTSModel)
			if err != nil {
				return err
			}
			srv.Speech = tts.NewService(voice, 256)
		} else {
			slog.Warn("VOICE_DISABLED: no voice configured")
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			engine.Listen(ctx, pool, nil, hub.Publish)
		}()
		httpSrv := &http.Server{Addr: cfg.Addr, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
		go func() {
			<-ctx.Done()
			c, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			_ = httpSrv.Shutdown(c)
		}()
		slog.Info("listening", "addr", cfg.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	<-ctx.Done()
	wg.Wait()
	return nil
}

func startWorkers(ctx context.Context, wg *sync.WaitGroup, cfg config.Config, store *engine.Store, model *engine.Model, chats *chatstore.Store, svc *dating.Service, bill *billing.Service, mailer mail.Mailer) {
	wake := make(chan struct{}, 16)
	turnWake := make(chan struct{}, 16)
	spawn := func(f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}
	spawn(func() { engine.Listen(ctx, store.Pool, wake, nil) })
	spawn(func() { listenTurns(ctx, store, turnWake) })
	planner := &engine.Planner{Store: store, Model: model, LLM: cfg.LLMModel}
	for i := 0; i < cfg.WorkerCount; i++ {
		w := &engine.Worker{ID: engine.NewWorkerID(), Store: store, Model: model, Planner: planner, LLM: cfg.LLMModel, Mailer: mailer, Lease: cfg.LeaseDuration, Wake: wake}
		spawn(func() { w.Run(ctx) })
	}
	for i := 0; i < cfg.TurnWorkers; i++ {
		r := &agent.Runner{ID: engine.NewWorkerID(), Pool: store.Pool, Chats: chats, Model: model, Dating: svc, LLM: cfg.LLMModel, FastLLM: cfg.LLMFastModel, Lease: cfg.TurnLease, Wake: turnWake, Billing: bill}
		spawn(func() { r.Run(ctx) })
	}
	sched := &engine.Scheduler{Store: store, Model: model, FastLLM: cfg.LLMFastModel, Mailer: mailer, PublicOrigin: cfg.PublicOrigin, Cron: workflows.Cron}
	spawn(func() { sched.Run(ctx) })
	// Turn leases and push run on every worker pod: both are safe to run twice.
	spawn(func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		httpClient := &http.Client{Timeout: 15 * time.Second}
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := agent.ReclaimExpired(ctx, store.Pool); err != nil && ctx.Err() == nil {
					slog.Error("reclaim turns", "err", err)
				}
				notify.PushDue(ctx, store.Pool, cfg.ExpoPushURL, httpClient)
			}
		}
	})
}

// listenTurns wakes turn workers as soon as a message arrives.
func listenTurns(ctx context.Context, store *engine.Store, wake chan<- struct{}) {
	for ctx.Err() == nil {
		conn, err := store.Pool.Acquire(ctx)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		if _, err = conn.Exec(ctx, `LISTEN syn_turn`); err == nil {
			for {
				if _, err = conn.Conn().WaitForNotification(ctx); err != nil {
					break
				}
				select {
				case wake <- struct{}{}:
				default:
				}
			}
		}
		conn.Release()
		if ctx.Err() == nil {
			time.Sleep(time.Second)
		}
	}
}
