package testing

import (
	"context"
	"fmt"

	"github.com/shah-dhwanil/tasker/internal/app"
)

var services *app.Services

func Setup()(func()){
	s,err := app.NewServices()
	if err != nil {
		panic(err)
	}
	services = s
	return func(){
		ctx:= context.Background()
		_,err:=s.DB().Exec(ctx,"DROP SCHEMA tasker CASCADE;")
		if err != nil {
			panic(fmt.Errorf("Error while droping tasker schema:%w",err))
		}
		_,err = s.DB().Exec(ctx,"DROP TABLE schema_version;")
		if err != nil {
			panic(fmt.Errorf("Error while droping schema_version table:%w",err))
		}
	}
}

func Services() *app.Services {
	if services == nil {
		panic("Services not initialized. Call Setup() first.")
	}
	return services
}