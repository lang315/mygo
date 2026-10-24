# ImageList

`ImageList(size)` keeps images by key, as WinForms' `ImageList` does, to share
between the controls that show images. It is not a control: make one, add the
images, and give the controls what it returns. `size` is the side, in DIPs, the
images are shown at.

```go
icons := widget.ImageList(16)
icons.Add("folder", folder16, folder32) // PNG, JPEG, GIF, WebP or BMP; one size or several
icons.Add("file", file16, file32)

tree := widget.TreeView(key, text).SetImage(widget.ImagesBy(icons, func(n *Node) string { return n.Icon }))
list := widget.ListBox(key, text).SetImage(widget.ImagesBy(icons, func(f *File) string { return f.Kind }))
menu := widget.MenuItem{Text: "Open", IconPNG: icons.PNG("folder")}
```

- `Add(key, data...)` decodes the images and keeps them under `key`, in place of
  those it had; it fails and changes nothing when there is none or one does not
  decode. `AddImage(key, imgs...)` takes `image.Image`s. The images of one key are
  the same picture at different sizes.
- `Has`, `Len`, `Keys` (in the order first added), `Remove(keys...)`, `Clear`.
- `Image(key)` is the bitmap of the largest size, which `ui` draws smaller
  smoothly (it keeps halved copies); `ImageAt(key, scale)` is the smallest size
  with a pixel for each pixel of `Size` at the display's scale (`ImageAt(k, 2)`
  of 16, 32 and 48 pixels is the 32), else the largest. `PNG(key)` is the
  largest as PNG data, for `MenuItem.IconPNG`.
- `ImagesBy(list, key)` is the function a control's `SetImage` takes: the image of
  the item whose key `key` returns, nil for none. Controls call it with the
  package's lock held, so inside such a callback use `ImagesBy`, not `Image`.
- Every method is safe from any goroutine.

Not here: WinForms' `ColorDepth` and `TransparentColor` (images keep their own
alpha), and an image index in place of a key. The controls that take a
`func(*T) *ui.Bitmap` need nothing else; a control that takes only a bitmap
takes `Image(key)`.
