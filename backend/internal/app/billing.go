package app

import (
	"context"
	"database/sql"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	notificationcore "github.com/TokenFlux/TokenRouter/internal/notification"
	"github.com/TokenFlux/TokenRouter/internal/promotion"
	settingscore "github.com/TokenFlux/TokenRouter/internal/settings"

	"github.com/TokenFlux/TokenRouter/internal/batchimage"

	batchpostgres "github.com/TokenFlux/TokenRouter/internal/batchimage/postgres"
	"github.com/TokenFlux/TokenRouter/internal/creative"

	creativepostgres "github.com/TokenFlux/TokenRouter/internal/creative/postgres"

	paymentpostgres "github.com/TokenFlux/TokenRouter/internal/payment/postgres"

	dbent "github.com/TokenFlux/TokenRouter/ent"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"

	"github.com/TokenFlux/TokenRouter/internal/billing"

	billingpostgres "github.com/TokenFlux/TokenRouter/internal/billing/postgres"

	"github.com/TokenFlux/TokenRouter/internal/config"

	identitypostgres "github.com/TokenFlux/TokenRouter/internal/identity/postgres"

	"github.com/TokenFlux/TokenRouter/internal/infra/telemetry/logging"

	"github.com/TokenFlux/TokenRouter/internal/pkg/timezone"

	routingpostgres "github.com/TokenFlux/TokenRouter/internal/routing/postgres"

	schedulerpostgres "github.com/TokenFlux/TokenRouter/internal/scheduler/postgres"

	"github.com/google/uuid"
)

func billingEligibilityOptions(c *config.Config, calendar timezone.Calendar) billing.EligibilityOptions {
	return billing.EligibilityOptions{Dates: billing.DateRuntime{Now: time.Now, Calendar: &calendar}, Billing: billing.BillingOptions{MinimumBalanceReserve: c.Billing.MinimumBalanceReserve, CircuitBreaker: billing.CircuitBreakerOptions{Enabled: c.Billing.CircuitBreaker.Enabled, FailureThreshold: c.Billing.CircuitBreaker.FailureThreshold, ResetTimeoutSeconds: c.Billing.CircuitBreaker.ResetTimeoutSeconds, HalfOpenRequests: c.Billing.CircuitBreaker.HalfOpenRequests}}}
}

// provideBillingEligibility 绑定余额与 Key 限额准入的共享缓存。
func provideBillingEligibility(cache billing.BillingCache, users *identitypostgres.UserStore, keys apikey.APIKeyRepository, cfg *config.Config, tasks *lifecycle.Tasks, calendar timezone.Calendar) *billing.Eligibility {
	options := billingEligibilityOptions(cfg, calendar)
	return billing.NewEligibility(cache, billingIdentityUsers{Repository: users}, keys, func() billing.EligibilityOptions { return options }, logging.LegacyPrintf, func(name string, fn func()) { tasks.Go(name, fn) })
}

func provideBillingSubscriptions(groups *routingpostgres.GroupStore, repo billing.UserSubscriptionRepository, client *dbent.Client, calendar timezone.Calendar) *billing.SubscriptionService {
	return billing.NewSubscriptionService(billingGroups{Repository: groups}, repo, billingpostgres.NewSubscriptionMutations(client), billing.DateRuntime{Now: time.Now, Calendar: &calendar})
}

func provideSettlementStore(db *sql.DB, calendar timezone.Calendar) *billingpostgres.SettlementStore {
	return billingpostgres.NewSettlementStore(db, calendar, schedulerpostgres.EnqueueProviderQuotaChangedInTx, billingpostgres.TaskProjectionFactories{
		creative.FundingScope: func(tx *sql.Tx, ref billing.TaskReference) billingpostgres.TaskProjection {
			return creativepostgres.NewFundingParticipant(tx, ref.ID)
		},
		batchimage.FundingScope: func(tx *sql.Tx, ref billing.TaskReference) billingpostgres.TaskProjection {
			return batchpostgres.NewFundingParticipant(tx, ref.ID)
		},
	})
}

func provideBillingFunds(store *billingpostgres.SettlementStore) *billing.Funds {
	return billing.NewFunds(store)
}

func provideBillingRedeem(repo billing.RedeemCodeRepository, users *identitypostgres.UserStore, subs *billing.SubscriptionService, cache billing.RedeemCache, eligibility *billing.Eligibility, client *dbent.Client, auth apikey.APIKeyAuthCacheInvalidator, affiliate *promotion.AffiliateService, tasks *lifecycle.Tasks) *billing.RedeemService {
	return billing.NewRedeemService(repo, billingIdentityUsers{Repository: users}, subs, cache, eligibility, billingpostgres.NewRedeemMutations(client, billingpostgres.RedeemWriters{Balances: billingpostgres.NewBalanceStore(client), Concurrency: identitypostgres.NewConcurrencyStore(client)}), auth, affiliate, billing.RedeemRuntime{Now: time.Now, Observe: logging.LegacyPrintf, Background: func(name string, fn func()) { tasks.Go(name, fn) }})
}

func provideRedeemAdministration(repo billing.RedeemCodeRepository, client *dbent.Client) *billing.RedeemAdmin {
	return billing.NewRedeemAdmin(repo, billingpostgres.NewRedeemAdministrationMutations(client), time.Now)
}

func provideBalanceAdjuster(client *dbent.Client) billing.BalanceAdjuster {
	return billingpostgres.NewBalanceStore(client)
}

func provideBillingPlans(client *dbent.Client, orders *paymentpostgres.InstanceStore) *billing.Plans {
	return billing.NewPlans(billingpostgres.NewPlanStore(client), orders)
}

// provideSubscriptionExpiry 注入旧通知与锁策略，构造期间不启动后台任务。
func provideSubscriptionExpiry(repo billing.UserSubscriptionRepository, settings settingscore.Repository, notification *notificationcore.NotificationEmailService, lock provider.CNMonitorLeader, db *sql.DB) *billing.SubscriptionExpiryService {
	return billing.NewSubscriptionExpiryService(repo, billing.ExpiryOptions{Interval: time.Minute, Owner: uuid.NewString(), Now: time.Now, Observe: func(format string, args ...any) { logging.LegacyPrintf("service.subscription_expiry", format, args...) }, Settings: settings, Notifier: expiryNotifications{Service: notification}, Lease: func(ctx context.Context, key, owner string, ttl time.Duration) (func(), bool) {
		return provider.AcquireSingletonLease(ctx, lock, databaseAdvisoryLease(db), key, owner, ttl)
	}})
}
