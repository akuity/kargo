# label requirement construction

In `newListOptionsForLabelSelector`, I use a helper func called `labelOpToSelectionOp` to convert `metav1.LabelSelectorOperator` into a `selection.Operator`.

This is because `labels.NewRequirement` (unintuitively?) requires a `selection.Operator` instead.

I could change `kargoapi.ConditionSelectorOperator.Operator` type to a `selection.Operator` instead but it supports more operators than we defined in the spec. However, I think we could validate this with `kubebuilder` enum validation.

However, the `selection.Operator` type values differ from the spec:

selection.Oerator

```
	DoesNotExist Operator = "!"
	Equals       Operator = "="
	DoubleEquals Operator = "=="
	In           Operator = "in"
	NotEquals    Operator = "!="
	NotIn        Operator = "notin"
	Exists       Operator = "exists"
	GreaterThan  Operator = "gt"
	LessThan     Operator = "lt"
```

Spec:

```
    Equals: Exact match (for conditions)
    NotEquals: Not exact match (for conditions)
    In: Value must be in the provided list
    NotIn: Value must not be in the provided list
    Exists: Key must exist (values ignored)
    DoesNotExist: Key must not exist (values ignored)
```

Expr-lang:



