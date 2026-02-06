// Package xmpp provides a comprehensive, production-grade XMPP library for Go.
//
// It supports both client and server roles, is fully extensible via a plugin
// architecture, and aims to cover every modern XMPP feature including OMEMO
// encryption, MUC, PubSub, Jingle, and more.
//
// The library is organized into several layers:
//
//   - Core: JID parsing, XML streaming, stanza types, transport abstractions
//   - Session: Stream negotiation, SASL, TLS, resource binding, stanza routing
//   - Client/Server: High-level APIs for building XMPP clients and servers
//   - Plugin System: Extensible architecture for XEP implementations
//   - Plugins: Ready-to-use implementations of 50+ XEPs
//
// Basic client usage:
//
//	client, err := xmpp.NewClient(
//	    jid.MustParse("user@example.com"),
//	    "password",
//	    xmpp.WithPlugins(
//	        disco.New(),
//	        roster.New(),
//	        carbons.New(),
//	    ),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	if err := client.Connect(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
package xmpp
