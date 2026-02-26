package main

import (
	"fmt"
	"sort"

	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/plugins/avatar"
	"github.com/meszmate/xmpp-go/plugins/blocking"
	"github.com/meszmate/xmpp-go/plugins/bob"
	"github.com/meszmate/xmpp-go/plugins/bookmarks"
	"github.com/meszmate/xmpp-go/plugins/caps"
	"github.com/meszmate/xmpp-go/plugins/carbons"
	"github.com/meszmate/xmpp-go/plugins/chatmarkers"
	"github.com/meszmate/xmpp-go/plugins/chatstates"
	"github.com/meszmate/xmpp-go/plugins/commands"
	"github.com/meszmate/xmpp-go/plugins/correction"
	"github.com/meszmate/xmpp-go/plugins/csi"
	"github.com/meszmate/xmpp-go/plugins/delay"
	"github.com/meszmate/xmpp-go/plugins/dialback"
	"github.com/meszmate/xmpp-go/plugins/disco"
	"github.com/meszmate/xmpp-go/plugins/extdisco"
	"github.com/meszmate/xmpp-go/plugins/filetransfer"
	"github.com/meszmate/xmpp-go/plugins/form"
	"github.com/meszmate/xmpp-go/plugins/forward"
	"github.com/meszmate/xmpp-go/plugins/hash"
	"github.com/meszmate/xmpp-go/plugins/hints"
	"github.com/meszmate/xmpp-go/plugins/ibb"
	"github.com/meszmate/xmpp-go/plugins/jingle"
	"github.com/meszmate/xmpp-go/plugins/lastactivity"
	"github.com/meszmate/xmpp-go/plugins/mam"
	"github.com/meszmate/xmpp-go/plugins/mix"
	"github.com/meszmate/xmpp-go/plugins/moderation"
	"github.com/meszmate/xmpp-go/plugins/muc"
	"github.com/meszmate/xmpp-go/plugins/omemo"
	"github.com/meszmate/xmpp-go/plugins/oob"
	"github.com/meszmate/xmpp-go/plugins/ping"
	"github.com/meszmate/xmpp-go/plugins/presence"
	"github.com/meszmate/xmpp-go/plugins/pubsub"
	"github.com/meszmate/xmpp-go/plugins/push"
	"github.com/meszmate/xmpp-go/plugins/reactions"
	"github.com/meszmate/xmpp-go/plugins/receipts"
	"github.com/meszmate/xmpp-go/plugins/register"
	"github.com/meszmate/xmpp-go/plugins/retraction"
	"github.com/meszmate/xmpp-go/plugins/roster"
	"github.com/meszmate/xmpp-go/plugins/rsm"
	"github.com/meszmate/xmpp-go/plugins/sasl2"
	"github.com/meszmate/xmpp-go/plugins/sm"
	"github.com/meszmate/xmpp-go/plugins/socks5"
	"github.com/meszmate/xmpp-go/plugins/stanzaid"
	"github.com/meszmate/xmpp-go/plugins/styling"
	"github.com/meszmate/xmpp-go/plugins/time"
	"github.com/meszmate/xmpp-go/plugins/upload"
	"github.com/meszmate/xmpp-go/plugins/vcard"
	"github.com/meszmate/xmpp-go/plugins/version"
)

func pluginRegistry(cfg Config) map[string]func() plugin.Plugin {
	return map[string]func() plugin.Plugin{
		"avatar":       func() plugin.Plugin { return avatar.New() },
		"blocking":     func() plugin.Plugin { return blocking.New() },
		"bob":          func() plugin.Plugin { return bob.New() },
		"bookmarks":    func() plugin.Plugin { return bookmarks.New() },
		"caps":         func() plugin.Plugin { return caps.New(cfg.CapsNode) },
		"carbons":      func() plugin.Plugin { return carbons.New() },
		"chatmarkers":  func() plugin.Plugin { return chatmarkers.New() },
		"chatstates":   func() plugin.Plugin { return chatstates.New() },
		"commands":     func() plugin.Plugin { return commands.New() },
		"correction":   func() plugin.Plugin { return correction.New() },
		"csi":          func() plugin.Plugin { return csi.New() },
		"delay":        func() plugin.Plugin { return delay.New() },
		"dialback":     func() plugin.Plugin { return dialback.New() },
		"disco":        func() plugin.Plugin { return disco.New() },
		"extdisco":     func() plugin.Plugin { return extdisco.New() },
		"filetransfer": func() plugin.Plugin { return filetransfer.New() },
		"form":         func() plugin.Plugin { return form.New() },
		"forward":      func() plugin.Plugin { return forward.New() },
		"hash":         func() plugin.Plugin { return hash.New() },
		"hints":        func() plugin.Plugin { return hints.New() },
		"ibb":          func() plugin.Plugin { return ibb.New() },
		"jingle":       func() plugin.Plugin { return jingle.New() },
		"lastactivity": func() plugin.Plugin { return lastactivity.New() },
		"mam":          func() plugin.Plugin { return mam.New() },
		"mix":          func() plugin.Plugin { return mix.New() },
		"moderation":   func() plugin.Plugin { return moderation.New() },
		"muc":          func() plugin.Plugin { return muc.New() },
		"oob":          func() plugin.Plugin { return oob.New() },
		"omemo":        func() plugin.Plugin { return omemo.New(cfg.OMEMODeviceID) },
		"ping":         func() plugin.Plugin { return ping.New() },
		"presence":     func() plugin.Plugin { return presence.New() },
		"pubsub":       func() plugin.Plugin { return pubsub.New() },
		"push":         func() plugin.Plugin { return push.New() },
		"reactions":    func() plugin.Plugin { return reactions.New() },
		"receipts":     func() plugin.Plugin { return receipts.New() },
		"register":     func() plugin.Plugin { return register.New() },
		"retraction":   func() plugin.Plugin { return retraction.New() },
		"roster":       func() plugin.Plugin { return roster.New() },
		"rsm":          func() plugin.Plugin { return rsm.New() },
		"sasl2":        func() plugin.Plugin { return sasl2.New() },
		"sm":           func() plugin.Plugin { return sm.New() },
		"socks5":       func() plugin.Plugin { return socks5.New() },
		"stanzaid":     func() plugin.Plugin { return stanzaid.New() },
		"styling":      func() plugin.Plugin { return styling.New() },
		"time":         func() plugin.Plugin { return time.New() },
		"upload":       func() plugin.Plugin { return upload.New() },
		"vcard":        func() plugin.Plugin { return vcard.New() },
		"version":      func() plugin.Plugin { return version.New(cfg.VersionName, cfg.VersionString) },
	}
}

func buildPlugins(cfg Config) ([]plugin.Plugin, error) {
	reg := pluginRegistry(cfg)
	if len(cfg.Plugins) == 0 {
		return nil, nil
	}

	if len(cfg.Plugins) == 1 && cfg.Plugins[0] == "all" {
		keys := make([]string, 0, len(reg))
		for k := range reg {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		plugins := make([]plugin.Plugin, 0, len(keys))
		for _, k := range keys {
			plugins = append(plugins, reg[k]())
		}
		return plugins, nil
	}

	plugins := make([]plugin.Plugin, 0, len(cfg.Plugins))
	for _, name := range cfg.Plugins {
		ctor, ok := reg[name]
		if !ok {
			return nil, fmt.Errorf("unknown plugin: %s", name)
		}
		plugins = append(plugins, ctor())
	}
	return plugins, nil
}
