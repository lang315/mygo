# BreadCrumb

`BreadCrumb(items...)` is a path of items separated by chevrons, as
DevExpress's `BreadCrumbEdit` used for navigation (it is not editable). The
items but the last are links, which a click or Enter chooses; the last is
where the path is. Long items shorten with an ellipsis as room runs short. It
is [`ui.Breadcrumbs`](../breadcrumbs.md) underneath.

```go
path := []string{"C:", "Users", "ada"}
var crumb *widget.BreadCrumbWidget
crumb = widget.BreadCrumb(path...).OnSelect(func(i int) {
	path = path[:i+1]
	crumb.SetItems(path...)
})
```

- `SetItems(items...)` replaces the items. The items are copied.
- `OnSelect(fn)` runs on the event goroutine with the index of the item the
  user chose, each time, the same item again too. The app usually trims its
  path to `[:i+1]` and sets the items again.

Assistive technology sees links in a navigation landmark, and the last item
as the current page.
