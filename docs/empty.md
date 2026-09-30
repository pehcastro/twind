# Empty

What a list or a page shows before it has anything.

<Preview name="empty-demo" />

## Usage

```go
ui.Empty(
	ui.EmptyHeader(
		ui.EmptyMedia(ui.Icon, twi.Text("▣")),
		ui.EmptyTitle(twi.Text("No projects yet")),
		ui.EmptyDescription(twi.Text("Start by creating your first one.")),
	),
	ui.EmptyContent(ui.Button(ui.Default, ui.SizeDefault, twi.Text("Create project"))),
)
```

## API reference

<Props of="Empty" />
