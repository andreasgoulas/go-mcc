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

func parseColor(arg string) (c color.RGBA, err error) {
	c.A = 0xff
	_, err = fmt.Sscanf(arg, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	return
}

func envWeather(config *gomcc.EnvConfig, arg string) bool {
	switch strings.ToLower(arg) {
	case "sun":
		config.Weather = gomcc.WeatherSunny
	case "rain":
		config.Weather = gomcc.WeatherRaining
	case "snow":
		config.Weather = gomcc.WeatherSnowing
	default:
		return false
	}

	return true
}

func envSideBlock(config *gomcc.EnvConfig, arg string) bool {
	if v, err := strconv.ParseUint(arg, 10, 64); err != nil || v > gomcc.BlockMax {
		return false
	} else {
		config.SideBlock = byte(v)
		return true
	}
}

func envEdgeBlock(config *gomcc.EnvConfig, arg string) bool {
	if v, err := strconv.ParseUint(arg, 10, 64); err != nil || v > gomcc.BlockMax {
		return false
	} else {
		config.EdgeBlock = byte(v)
		return true
	}
}

func envEdgeHeight(config *gomcc.EnvConfig, arg string) bool {
	if v, err := strconv.ParseUint(arg, 10, 64); err != nil {
		return false
	} else {
		config.EdgeHeight = int(v)
		return true
	}
}

func envCloudHeight(config *gomcc.EnvConfig, arg string) bool {
	if v, err := strconv.ParseUint(arg, 10, 64); err != nil {
		return false
	} else {
		config.CloudHeight = int(v)
		return true
	}
}

func envMaxViewDistance(config *gomcc.EnvConfig, arg string) bool {
	if v, err := strconv.ParseUint(arg, 10, 64); err != nil {
		return false
	} else {
		config.MaxViewDistance = int(v)
		return true
	}
}

func envCloudSpeed(config *gomcc.EnvConfig, arg string) bool {
	if v, err := strconv.ParseFloat(arg, 64); err != nil {
		return false
	} else {
		config.CloudSpeed = v
		return true
	}
}

func envWeatherSpeed(config *gomcc.EnvConfig, arg string) bool {
	if v, err := strconv.ParseFloat(arg, 64); err != nil {
		return false
	} else {
		config.WeatherSpeed = v
		return true
	}
}

func envWeatherFade(config *gomcc.EnvConfig, arg string) bool {
	if v, err := strconv.ParseFloat(arg, 64); err != nil || v < 0.0 {
		return false
	} else {
		config.WeatherFade = v
		return true
	}
}

func envExpFog(config *gomcc.EnvConfig, arg string) bool {
	if v, err := strconv.ParseBool(arg); err != nil {
		return false
	} else {
		config.ExpFog = v
		return true
	}
}

func envSideOffset(config *gomcc.EnvConfig, arg string) bool {
	if v, err := strconv.ParseInt(arg, 10, 64); err != nil {
		return false
	} else {
		config.SideOffset = int(v)
		return true
	}
}

func envSkyColor(config *gomcc.EnvConfig, arg string) bool {
	if v, err := parseColor(arg); err != nil {
		return false
	} else {
		config.SkyColor = v
		return true
	}
}

func envCloudColor(config *gomcc.EnvConfig, arg string) bool {
	if v, err := parseColor(arg); err != nil {
		return false
	} else {
		config.CloudColor = v
		return true
	}
}

func envFogColor(config *gomcc.EnvConfig, arg string) bool {
	if v, err := parseColor(arg); err != nil {
		return false
	} else {
		config.FogColor = v
		return true
	}
}

func envAmbientColor(config *gomcc.EnvConfig, arg string) bool {
	if v, err := parseColor(arg); err != nil {
		return false
	} else {
		config.AmbientColor = v
		return true
	}
}

func envDiffuseColor(config *gomcc.EnvConfig, arg string) bool {
	if v, err := parseColor(arg); err != nil {
		return false
	} else {
		config.DiffuseColor = v
		return true
	}
}

var envOptions = map[string]struct {
	Set  func(config *gomcc.EnvConfig, arg string) bool
	Mask uint32
}{
	"weather":         {envWeather, gomcc.EnvPropWeather},
	"sideblock":       {envSideBlock, gomcc.EnvPropSideBlock},
	"edgeblock":       {envEdgeBlock, gomcc.EnvPropEdgeBlock},
	"edgeheight":      {envEdgeHeight, gomcc.EnvPropEdgeHeight},
	"cloudheight":     {envCloudHeight, gomcc.EnvPropCloudHeight},
	"maxviewdistance": {envMaxViewDistance, gomcc.EnvPropMaxViewDistance},
	"cloudspeed":      {envCloudSpeed, gomcc.EnvPropCloudSpeed},
	"weatherspeed":    {envWeatherSpeed, gomcc.EnvPropWeatherSpeed},
	"weatherfade":     {envWeatherFade, gomcc.EnvPropWeatherFade},
	"expfog":          {envExpFog, gomcc.EnvPropExpFog},
	"sideoffset":      {envSideOffset, gomcc.EnvPropSideOffset},
	"skycolor":        {envSkyColor, gomcc.EnvPropSkyColor},
	"cloudcolor":      {envCloudColor, gomcc.EnvPropCloudColor},
	"fogcolor":        {envFogColor, gomcc.EnvPropFogColor},
	"ambientcolor":    {envAmbientColor, gomcc.EnvPropAmbientColor},
	"diffusecolor":    {envDiffuseColor, gomcc.EnvPropDiffuseColor},
}

func (plugin *Plugin) handleEnv(sender gomcc.CommandSender, command *gomcc.Command, message string) {
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
			level.EnvConfig = level.DefaultEnvConfig()
			level.SendEnvConfig(gomcc.EnvPropAll)
			return
		}

	case 2:
		opt, ok := envOptions[strings.ToLower(args[0])]
		if !ok {
			sender.SendMessage("Unknown option")
			return
		}

		level := player.Level()
		if opt.Set(&level.EnvConfig, args[1]) {
			level.SendEnvConfig(opt.Mask)
		} else {
			sender.SendMessage("Invalid value")
		}

		return
	}

	sender.SendMessage("Usage: " + command.Name + " <option> <value>")
	return
}
