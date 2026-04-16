package rbac

import (
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

func InitEnforcer(db *gorm.DB, modelPath string) (*casbin.Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDBUseTableName(db, "", "casbin_rule")
	if err != nil {
		return nil, err
	}
	if err := normalizePolicyScopes(db); err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, err
	}
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, err
	}
	if err := ensureAdminPolicy(enforcer); err != nil {
		return nil, err
	}
	if err := enforcer.SavePolicy(); err != nil {
		return nil, err
	}
	return enforcer, nil
}

func ensureAdminPolicy(enforcer *casbin.Enforcer) error {
	hasGrouping, err := enforcer.HasGroupingPolicy("admin", "admin")
	if err != nil {
		return err
	}
	if !hasGrouping {
		if _, err := enforcer.AddGroupingPolicy("admin", "admin"); err != nil {
			return err
		}
	}
	hasPolicy, err := enforcer.HasPolicy("admin", "/api/v1/*", ".*", "*", "*", "*")
	if err != nil {
		return err
	}
	if !hasPolicy {
		if _, err := enforcer.AddPolicy("admin", "/api/v1/*", ".*", "*", "*", "*"); err != nil {
			return err
		}
	}
	return nil
}

func normalizePolicyScopes(db *gorm.DB) error {
	updates := []string{"v3", "v4", "v5"}
	for _, field := range updates {
		if err := db.Table("casbin_rule").
			Where("ptype = ? AND ("+field+" IS NULL OR "+field+" = '')", "p").
			Update(field, "*").Error; err != nil {
			return err
		}
	}
	return nil
}
