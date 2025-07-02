package data

import (
	"github.com/arnokay/arnobot-shared/data"
	"github.com/scorfly/gokick"
)

func GetChatterRole(badges []gokick.Badge) data.ChatterRole {
	var role = data.ChatterPleb
	for _, badge := range badges {
		if badge.Type == "subscriber" && role < data.ChatterSub {
			role = data.ChatterSub
		}
		if (badge.Type == "og" || badge.Type == "vip") && role < data.ChatterVIP {
			role = data.ChatterVIP
		}
		if badge.Type == "moderator" && role < data.ChatterModerator {
			role = data.ChatterModerator
		}
		if badge.Type == "broadcaster" && role < data.ChatterBroadcaster {
			role = data.ChatterBroadcaster
		}
	}

	return role
}
