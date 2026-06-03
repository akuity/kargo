# Auto-Promotion Hold UX Demo

This is a small set of demo resources for discussion on auto-promotion-hold UX.

```shell
./hack/testing/auto-promotion-hold/apply.sh
```

The demo seeds two Stages:

- `single-origin-hold`: single-origin Stage with normal auto-promotion enabled.
- `multi-origin-holds`: multi-origin Stage with active auto-promotion holds for
  two origins.

Use `multi-origin-holds` to review how the DAG and list views show held origins.
The held-origin popover should list each origin independently and only resume
the selected origin.
