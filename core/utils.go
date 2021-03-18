package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/AndreasGoulas/go-mcc/mcc"
)

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func fmtDuration(t time.Duration) string {
	t = t.Round(time.Minute)
	d := t / (24 * time.Hour)
	t -= d * (24 * time.Hour)
	h := t / time.Hour
	t -= h * time.Hour
	m := t / time.Minute
	return fmt.Sprintf("%dd %dh %dm", d, h, m)
}

func parseCoord(arg string, curr float64) (float64, error) {
	if strings.HasPrefix(arg, "~") {
		value, err := strconv.Atoi(arg[1:])
		return curr + float64(value), err
	} else {
		value, err := strconv.Atoi(arg)
		return float64(value), err
	}
}

func parseColor(arg string) (c mcc.NullRGB, err error) {
	c.Valid = true
	_, err = fmt.Sscanf(arg, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	return
}

func envOption(option string, arg string, config *mcc.EnvConfig) int32 {
	switch strings.ToLower(option) {
	case "weather":
		switch strings.ToLower(arg) {
		case "sun":
			config.Weather = mcc.WeatherSunny
			return mcc.EnvPropWeather
		case "rain":
			config.Weather = mcc.WeatherRaining
			return mcc.EnvPropWeather
		case "snow":
			config.Weather = mcc.WeatherSnowing
			return mcc.EnvPropWeather
		}

	case "sideblock":
		if v, err := strconv.ParseUint(arg, 10, 64); err == nil && v <= mcc.BlockMax {
			config.SideBlock = byte(v)
			return mcc.EnvPropSideBlock
		}

	case "edgeblock":
		if v, err := strconv.ParseUint(arg, 10, 64); err == nil && v <= mcc.BlockMax {
			config.EdgeBlock = byte(v)
			return mcc.EnvPropEdgeBlock
		}

	case "edgeheight":
		if v, err := strconv.ParseUint(arg, 10, 64); err == nil {
			config.EdgeHeight = int(v)
			return mcc.EnvPropEdgeHeight
		}

	case "cloudheight":
		if v, err := strconv.ParseUint(arg, 10, 64); err == nil {
			config.CloudHeight = int(v)
			return mcc.EnvPropCloudHeight
		}

	case "maxviewdistance":
		if v, err := strconv.ParseUint(arg, 10, 64); err == nil {
			config.MaxViewDistance = int(v)
			return mcc.EnvPropMaxViewDistance
		}

	case "cloudspeed":
		if v, err := strconv.ParseFloat(arg, 64); err == nil {
			config.CloudSpeed = v
			return mcc.EnvPropCloudSpeed
		}

	case "weatherspeed":
		if v, err := strconv.ParseFloat(arg, 64); err == nil {
			config.WeatherSpeed = v
			return mcc.EnvPropWeatherSpeed
		}

	case "weatherfade":
		if v, err := strconv.ParseFloat(arg, 64); err == nil && v >= 0.0 {
			config.WeatherFade = v
			return mcc.EnvPropWeatherFade
		}

	case "expfog":
		if v, err := strconv.ParseBool(arg); err == nil {
			config.ExpFog = v
			return mcc.EnvPropExpFog
		}

	case "sideoffset":
		if v, err := strconv.ParseInt(arg, 10, 64); err == nil {
			config.SideOffset = int(v)
			return mcc.EnvPropSideOffset
		}

	case "skycolor":
		if v, err := parseColor(arg); err == nil {
			config.SkyColor = v
			return mcc.EnvPropSkyColor
		}

	case "cloudcolor":
		if v, err := parseColor(arg); err == nil {
			config.CloudColor = v
			return mcc.EnvPropCloudColor
		}

	case "fogcolor":
		if v, err := parseColor(arg); err == nil {
			config.FogColor = v
			return mcc.EnvPropFogColor
		}

	case "ambientcolor":
		if v, err := parseColor(arg); err == nil {
			config.AmbientColor = v
			return mcc.EnvPropAmbientColor
		}

	case "diffusecolor":
		if v, err := parseColor(arg); err == nil {
			config.DiffuseColor = v
			return mcc.EnvPropDiffuseColor
		}

	default:
		return 0
	}

	return -1
}

func parseMOTD(motd string, config *mcc.HackConfig) {
	for _, arg := range strings.Fields(motd) {
		if arg[0] == '+' || arg[0] == '-' {
			value := arg[0] == '+'
			switch strings.ToLower(arg[1:]) {
			case "fly":
				config.Flying = value
			case "noclip":
				config.NoClip = value
			case "speed":
				config.Speeding = value
			case "respawn":
				config.SpawnControl = value
			case "thirdperson":
				config.ThirdPersonView = value
			case "hax":
				config.Flying = value
				config.NoClip = value
				config.Speeding = value
				config.SpawnControl = value
				config.ThirdPersonView = value
			}
		} else if strings.HasPrefix(arg, "jumpheight=") {
			i := strings.IndexByte(arg, '=')
			if value, err := strconv.ParseFloat(arg[i+1:], 64); err == nil {
				config.JumpHeight = value
			}
		}
	}
}
