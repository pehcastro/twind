# Input OTP

A one-time code field, a box for each character.

<Preview name="input-otp-demo" />

## Usage

```go
code := ui.NewInputOTP(rt, 6)
code.OnChange = func(value string) { entered = value }
```

```go
code.Node(
	ui.InputOTPGroup(code.Slot(0), code.Slot(1), code.Slot(2)),
	ui.InputOTPSeparator(),
	ui.InputOTPGroup(code.Slot(3), code.Slot(4), code.Slot(5)),
)
```

It takes letters and digits up to its length; Backspace removes the last one.

## API reference

<Props of="InputOTP" />
