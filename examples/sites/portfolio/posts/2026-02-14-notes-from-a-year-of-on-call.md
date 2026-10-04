# Notes from a year of on-call

<Meta date="2026-02-14" tags="ops, people" />

A year ago I joined the on-call rotation for the first time. Here is what I wish someone had told me on day one.

1. Most pages are not emergencies. Learn which ones are, and say so in the alert's title.
2. Write down what you did while you do it. Tomorrow you will not remember.
3. Sleep is part of the job. Hand off when you are tired, not when you are done.

The runbook I keep now fits on one screen:

```bash
kubectl get pods -n checkout --field-selector=status.phase!=Running
kubectl logs -n checkout deploy/api --since=15m | grep -i error
```

None of this is new. It took a year to believe it.
