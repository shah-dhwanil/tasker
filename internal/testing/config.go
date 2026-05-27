package testing

import (
	"strings"

	"github.com/joho/godotenv"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/v2"
	"github.com/shah-dhwanil/tasker/internal/config"
)

func init(){
	godotenv.Overload("../../.env.test")
	cfg:= config.GetConfig()
	koanfClient := koanf.New(".")
	if err := koanfClient.Load(env.Provider(".", env.Opt{
		Prefix: "TASKER_",
		TransformFunc: func(k, v string) (string, any) {
			return strings.ToLower(strings.TrimPrefix(k, "TASKER_")), v
		},
	}), nil); err != nil {
		panic("Error in loading Environment variables")
	}
	koanfClient.Unmarshal("", cfg)
}