// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/structinf/Go-MCC/gomcc"
)

func envBlock(mask uint32, v *byte, def byte, arg string) uint32 {
	if arg == "reset" {
		*v = def
	} else {
		if tmp, err := strconv.ParseUint(arg, 10, 8); err != nil || tmp >= gomcc.BlockMax {
			return 0
		} else {
			*v = byte(tmp)
		}
	}

	return mask
}

func envUint(mask uint32, v *uint, def uint, arg string) uint32 {
	if arg == "reset" {
		*v = def
	} else {
		if tmp, err := strconv.ParseUint(arg, 10, 64); err != nil {
			return 0
		} else {
			*v = uint(tmp)
		}
	}

	return mask
}

func envInt(mask uint32, v *int, def int, arg string) uint32 {
	if arg == "reset" {
		*v = def
	} else {
		if tmp, err := strconv.ParseInt(arg, 10, 64); err != nil {
			return 0
		} else {
			*v = int(tmp)
		}
	}

	return mask
}

func envFloat(mask uint32, v *float64, def float64, arg string) uint32 {
	if arg == "reset" {
		*v = def
	} else {
		if tmp, err := strconv.ParseFloat(arg, 64); err != nil {
			return 0
		} else {
			*v = tmp
		}
	}

	return mask
}

func envBool(mask uint32, v *bool, def bool, arg string) uint32 {
	if arg == "reset" {
		*v = def
	} else {
		if tmp, err := strconv.ParseBool(arg); err != nil {
			return 0
		} else {
			*v = tmp
		}
	}

	return mask
}

func envColor(mask uint32, v *color.RGBA, def color.RGBA, arg string) uint32 {
	if arg == "reset" {
		*v = def
	} else {
		v.A = 0xff
		if _, err := fmt.Sscanf(arg, "#%02x%02x%02x", &v.R, &v.G, &v.B); err != nil {
			return 0
		}
	}

	return mask
}

func envWeather(config, def *gomcc.EnvConfig, arg string) uint32 {
	switch strings.ToLower(arg) {
	case "reset":
		config.Weather = def.Weather
	case "sun":
		config.Weather = gomcc.WeatherSunny
	case "rain":
		config.Weather = gomcc.WeatherRaining
	case "snow":
		config.Weather = gomcc.WeatherSnowing
	default:
		return 0
	}

	return gomcc.EnvPropWeather
}

func envSideBlock(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envBlock(gomcc.EnvPropSideBlock, &config.SideBlock, def.SideBlock, arg)
}

func envEdgeBlock(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envBlock(gomcc.EnvPropEdgeBlock, &config.EdgeBlock, def.EdgeBlock, arg)
}

func envEdgeHeight(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envUint(gomcc.EnvPropEdgeHeight, &config.EdgeHeight, def.EdgeHeight, arg)
}

func envCloudHeight(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envUint(gomcc.EnvPropCloudHeight, &config.CloudHeight, def.CloudHeight, arg)
}

func envMaxViewDistance(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envUint(gomcc.EnvPropMaxViewDistance, &config.MaxViewDistance, def.MaxViewDistance, arg)
}

func envCloudSpeed(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envFloat(gomcc.EnvPropCloudSpeed, &config.CloudSpeed, def.CloudSpeed, arg)
}

func envWeatherSpeed(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envFloat(gomcc.EnvPropWeatherSpeed, &config.WeatherSpeed, def.WeatherSpeed, arg)
}

func envWeatherFade(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envFloat(gomcc.EnvPropWeatherFade, &config.WeatherFade, def.WeatherFade, arg)
}

func envExpFog(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envBool(gomcc.EnvPropExpFog, &config.ExpFog, def.ExpFog, arg)
}

func envSideOffset(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envInt(gomcc.EnvPropSideOffset, &config.SideOffset, def.SideOffset, arg)
}

func envSkyColor(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envColor(gomcc.EnvPropSkyColor, &config.SkyColor, def.SkyColor, arg)
}

func envCloudColor(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envColor(gomcc.EnvPropCloudColor, &config.CloudColor, def.CloudColor, arg)
}

func envFogColor(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envColor(gomcc.EnvPropFogColor, &config.FogColor, def.FogColor, arg)
}

func envAmbientColor(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envColor(gomcc.EnvPropAmbientColor, &config.AmbientColor, def.AmbientColor, arg)
}

func envDiffuseColor(config, def *gomcc.EnvConfig, arg string) uint32 {
	return envColor(gomcc.EnvPropDiffuseColor, &config.DiffuseColor, def.DiffuseColor, arg)
}

type envFunc func(config, def *gomcc.EnvConfig, arg string) uint32

var envOptions = map[string]envFunc{
	"weather":         envWeather,
	"sideblock":       envSideBlock,
	"edgeblock":       envEdgeBlock,
	"edgeheight":      envEdgeHeight,
	"cloudheight":     envCloudHeight,
	"maxviewdistance": envMaxViewDistance,
	"cloudspeed":      envCloudSpeed,
	"weatherspeed":    envWeatherSpeed,
	"weatherfade":     envWeatherFade,
	"expfog":          envExpFog,
	"sideoffset":      envSideOffset,
	"skycolor":        envSkyColor,
	"cloudcolor":      envCloudColor,
	"fogcolor":        envFogColor,
	"ambientcolor":    envAmbientColor,
	"diffusecolor":    envDiffuseColor,
}

func (plugin *CorePlugin) handleEnv(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	player, ok := sender.(*gomcc.Player)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	args := strings.Fields(message)
	switch len(args) {
	case 1:
		if args[0] == "reset" {
			level := player.Level()
			level.EnvConfig = level.DefaultEnv()
			level.SendEnvConfig(gomcc.EnvPropAll)
			return
		}

	case 2:
		fn, ok := envOptions[strings.ToLower(args[0])]
		if !ok {
			sender.SendMessage("Unknown option")
			return
		}

		level := player.Level()
		defaultEnv := level.DefaultEnv()
		if mask := fn(&level.EnvConfig, &defaultEnv, args[1]); mask == 0 {
			sender.SendMessage("Invalid value")
			return
		} else {
			level.SendEnvConfig(mask)
		}

		return
	}

	sender.SendMessage("Usage: " + command.Name + " <option> <value>")
	return
}
