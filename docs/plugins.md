# Plugin Development Guide

## Plugin Interface

Every plugin implements the `plugin.Plugin` interface:

```go
type Plugin interface {
    Name() string
    Version() string
    Initialize(ctx context.Context, params plugin.InitParams) error
    Close() error
    Dependencies() []string
}
```

`InitParams` provides everything a plugin needs:

```go
type InitParams struct {
    SendRaw    func(ctx context.Context, data []byte) error
    SendElement func(ctx context.Context, v any) error
    State      func() uint32
    LocalJID   func() string
    RemoteJID  func() string
    Get        func(name string) (Plugin, bool)
    Storage    storage.Storage  // may be nil
}
```

## Creating a Plugin

```go
package myplugin

import (
    "context"

    "github.com/meszmate/xmpp-go/plugin"
)

type MyPlugin struct {
    params plugin.InitParams
}

func New() *MyPlugin {
    return &MyPlugin{}
}

func (p *MyPlugin) Name() string    { return "my-plugin" }
func (p *MyPlugin) Version() string { return "1.0.0" }

func (p *MyPlugin) Initialize(ctx context.Context, params plugin.InitParams) error {
    p.params = params
    return nil
}

func (p *MyPlugin) Close() error { return nil }
func (p *MyPlugin) Dependencies() []string { return nil }
```

## Dependencies

Return the names of plugins your plugin depends on from `Dependencies()`. The plugin manager performs topological sorting to ensure correct initialization order.

## Stream Features

Plugins can contribute stream features that are negotiated during connection setup. Return them from `StreamFeatures()`.

## Using Storage in Plugins

Plugins receive the configured storage backend through `params.Storage`. The recommended pattern is to check for the relevant sub-store during initialization and fall back to an in-memory data structure when storage is not configured:

```go
func (p *MyPlugin) Initialize(ctx context.Context, params plugin.InitParams) error {
    p.params = params
    if params.Storage != nil {
        p.store = params.Storage.VCardStore()
    }
    if p.store == nil {
        p.vcards = make(map[string][]byte) // in-memory fallback
    }
    return nil
}
```

All built-in stateful plugins (roster, blocking, vcard, MUC, MAM, PubSub, bookmarks) follow this pattern. Transient plugins (presence, disco, stream management, CSI, carbons, caps) do not use storage.
