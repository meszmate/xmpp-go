# Plugin Development Guide

## Plugin Interface

Every plugin implements the `plugin.Plugin` interface:

```go
type Plugin interface {
    Name() string
    Version() string
    Initialize(ctx context.Context, session *xmpp.Session) error
    Close() error
    StreamFeatures() []xmpp.StreamFeature
    MuxOptions() []mux.Option
    Dependencies() []string
}
```

## Creating a Plugin

```go
package myplugin

import (
    "context"

    xmpp "github.com/meszmate/xmpp-go"
    "github.com/meszmate/xmpp-go/plugin"
)

type MyPlugin struct {
    session *xmpp.Session
}

func New() *MyPlugin {
    return &MyPlugin{}
}

func (p *MyPlugin) Name() string    { return "my-plugin" }
func (p *MyPlugin) Version() string { return "1.0.0" }

func (p *MyPlugin) Initialize(ctx context.Context, session *xmpp.Session) error {
    p.session = session
    return nil
}

func (p *MyPlugin) Close() error { return nil }
func (p *MyPlugin) StreamFeatures() []xmpp.StreamFeature { return nil }
func (p *MyPlugin) MuxOptions() []xmpp.MuxOption { return nil }
func (p *MyPlugin) Dependencies() []string { return nil }
```

## Dependencies

Return the names of plugins your plugin depends on from `Dependencies()`. The plugin manager performs topological sorting to ensure correct initialization order.

## Stream Features

Plugins can contribute stream features that are negotiated during connection setup. Return them from `StreamFeatures()`.

## Stanza Handlers

Return `MuxOption` values from `MuxOptions()` to register stanza handlers with the session's multiplexer.
