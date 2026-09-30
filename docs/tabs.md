# Tabs

A set of layered sections of content, known as tab panels, that are displayed one at a time.

<Preview name="tabs-demo" />

## Usage

```go
tabs := ui.NewTabs(rt)
```

```go
tabs.Node(
	tabs.List(
		tabs.Trigger("account", twi.Text("Account")),
		tabs.Trigger("password", twi.Text("Password")),
	),
	tabs.Content("account", twi.Text("Make changes to your account here.")),
	tabs.Content("password", twi.Text("Change your password here.")),
)
```

The first trigger is selected until `Value` says otherwise. With the list focused, the Left and Right arrows, Home and End move between tabs.

## API reference

<Props of="Tabs" />
