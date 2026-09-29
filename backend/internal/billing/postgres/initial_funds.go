package postgres

import (
	dbent "github.com/TokenFlux/TokenRouter/ent"
	"github.com/TokenFlux/TokenRouter/internal/billing"
)

// ApplyInitialUserFunds 将初始资金加入同一条用户 INSERT，不另行提交或刷新缓存。
func ApplyInitialUserFunds(create *dbent.UserCreate, funds billing.InitialUserFunds) {
	create.SetBalance(funds.Balance)
}
