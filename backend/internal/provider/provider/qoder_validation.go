// 本文件把提供商凭据规则与 Qoder 站点交换连接起来，不持有授权缓存。
package provider

import (
	"context"
	"fmt"
	"strings"

	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
)

type qoderCredentialValidator struct {
	transport     QoderTransport
	profiles      *egressprovider.TLSProfiles
	validatePAT   func(context.Context, *provider.Record, string, *qoder.MachineIdentity) (*qoder.AuthIdentity, error)
	validateCNPAT func(context.Context, *provider.Record, string, *qoder.MachineIdentity, qoder.RequestDoer) (*qoder.AuthIdentity, error)
}

// CreateCredentialHooks 保留新建机器身份、编辑兼容和 PAT 校验的原有时机。
func CreateCredentialHooks(transport QoderTransport, profiles *egressprovider.TLSProfiles) provider.CreateCredentialHooks {
	return (&qoderCredentialValidator{transport: transport, profiles: profiles}).hooks()
}

func (v *qoderCredentialValidator) hooks() provider.CreateCredentialHooks {
	return provider.CreateCredentialHooks{
		Site: func(value *provider.Record) (string, error) {
			site, err := qoderSiteForRecord(value)
			return string(site), err
		},
		Prepare: func(value *provider.Record) {
			value.Credentials = provider.CloneValues(value.Credentials)
			ensureQoderMachineCredentials(value)
		},
		Validate:     v.validate,
		ValidateEdit: v.validateEdit,
	}
}

func (v *qoderCredentialValidator) validate(ctx context.Context, value *provider.Record) error {
	return v.validateEdit(ctx, value, false)
}

func (v *qoderCredentialValidator) validateEdit(ctx context.Context, value *provider.Record, deferPAT bool) error {
	if value != nil {
		value.Credentials = provider.CloneValues(value.Credentials)
		value.Extra = provider.CloneValues(value.Extra)
		// 邮箱仅补展示缺项，保留管理员明确值，不拿未验签声明作授权依据。
		if value.Platform == provider.PlatformOpenAI && value.Type == provider.ProviderTypeOAuth && strings.TrimSpace(value.GetCredential("email")) == "" {
			for _, key := range []string{"access_token", "id_token"} {
				if email := openai.TokenDisplayEmail(value.GetCredential(key)); email != "" {
					value.Credentials["email"] = email
					break
				}
			}
		}
	}
	return provider.ValidateQoderCredentials(ctx, value, deferPAT, provider.QoderCredentialValidation{
		Normalize: func(site, mode string) (string, error) {
			parsed, err := qoder.ParseSite(site)
			if err != nil {
				return "", err
			}
			_, err = qoder.ParseRefreshMode(mode)
			return string(parsed), err
		},
		ValidatePAT: func(ctx context.Context, value *provider.Record, site, pat string) error {
			machine := qoder.MachineForCredentials(QoderCredentialInput(value))
			doer := QoderRequestDoer(value, v.transport, v.profiles)
			if qoder.Site(site) == qoder.SiteCN {
				validate := v.validateCNPAT
				if validate == nil {
					validate = func(ctx context.Context, _ *provider.Record, pat string, machine *qoder.MachineIdentity, doer qoder.RequestDoer) (*qoder.AuthIdentity, error) {
						profile, err := qoder.ProfileForSite(qoder.SiteCN)
						if err != nil {
							return nil, err
						}
						identity, _, err := qoder.ExchangeQoderCN20PATContext(ctx, pat, machine, profile, doer)
						return identity, err
					}
				}
				if _, err := validate(ctx, value, pat, machine, doer); err != nil {
					return fmt.Errorf("validate qoder cn pat: %w", err)
				}
				return nil
			}
			validate := v.validatePAT
			if validate == nil || v.transport != nil {
				validate = func(ctx context.Context, _ *provider.Record, pat string, machine *qoder.MachineIdentity) (*qoder.AuthIdentity, error) {
					return qoder.ExchangePATContext(ctx, pat, machine, "", doer)
				}
			}
			if _, err := validate(ctx, value, pat, machine); err != nil {
				return fmt.Errorf("validate qoder pat: %w", err)
			}
			return nil
		},
	})
}

func qoderSiteForRecord(value *provider.Record) (qoder.Site, error) {
	if value == nil {
		return qoder.SiteGlobal, fmt.Errorf("qoder: provider is nil")
	}
	return qoder.ParseSite(value.GetCredential("site"))
}
