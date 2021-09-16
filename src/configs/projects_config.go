package configs

import (
	"github.com/spf13/viper"
)

type Conf struct {
	Projects []map[string]map[string]string
}

func GetProjectsConfig(path string) Conf {
	var projectsConfigs Conf
	var viperConfig = viper.New()
	viperConfig.SetConfigName("config")
	viperConfig.AddConfigPath(path + "/")
	viperConfig.SetConfigType("json")
	err := viperConfig.ReadInConfig()
	if err != nil {
		panic(err)
	}
	err = viperConfig.Unmarshal(&projectsConfigs)
	if err != nil {
		panic(err)
	}
	return projectsConfigs
}
