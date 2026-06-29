package server

import (
	"log"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

func SetupCasbin(db *gorm.DB) *casbin.Enforcer {

	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		log.Fatal(err)
	}

	casbinModel := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`
	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		log.Fatal(err)
	}

	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		log.Fatal(err)
	}

	err = enforcer.LoadPolicy()
	if err != nil {
		log.Fatal(err)
	}

	enforcer.EnableAutoSave(true)

	// Scaffolding: Allow "anon" user full access for now
	enforcer.AddPolicy("anon", "/*", "*")

	return enforcer
}
