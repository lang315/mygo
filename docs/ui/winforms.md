# WinForms catalogue

Every control and component of WinForms' toolbox, and where `ui/widget` stands
on it. Status: **done** (with the page that documents it), **pending** (what
and who), or **dropped** (why). This is the exit audit of Phase 2 of the
[roadmap](../roadmap.md); the names mygo uses are in
[widget.md](widget.md#names).

## Controls

| WinForms | Status | Where |
|---|---|---|
| Button | done | `Button`, `MenuButton`: [widget.md](widget.md) |
| CheckBox | done | `CheckBox`, `Toggle`, `Switch` |
| CheckedListBox | done | `CheckedListBox`: [ListBox](widget/listbox.md) |
| ComboBox | done | `ComboBox` (also editable) |
| DataGridView | done | [DataGridView](widget.md#datagridview) |
| DateTimePicker | done | [DateTimePicker](widget/date-time-picker.md) |
| DomainUpDown | done | [DomainUpDown](widget/domainupdown.md) |
| FlowLayoutPanel | done | `FlowPanel` |
| GroupBox | done | `GroupBox`, `Card` |
| Label | done | `Label` (`&` mnemonics with `SetUseMnemonic(true)`: [Mnemonics](widget/mnemonics.md)) |
| LinkLabel | done | `LinkLabel` |
| ListBox | done | [ListBox](widget/listbox.md) |
| ListView | done | [ListView](widget/listview.md) (item groups, `ListViewGroup`, are open: below) |
| MaskedTextBox | done | [MaskedTextBox](widget/maskedtextbox.md) |
| MonthCalendar | done | [DateTimePicker](widget/date-time-picker.md) |
| NumericUpDown | done | `NumericUpDown` |
| Panel | done | `VBox`, `HBox`, `Card`, `ScrollPanel` (AutoScroll) |
| PictureBox | done | `PictureBox` |
| ProgressBar | done | `ProgressBar` |
| PropertyGrid | dropped (Phase 2) | fhtool and the target apps edit settings in dialogs built from `Fields` and `Bind`; a reflection-driven grid can come later on request |
| RadioButton | done | `RadioButton`, `RadioGroup` |
| RichTextBox | done | [RichTextBox](widget/richtextbox.md) |
| SplitContainer | done | [SplitContainer](widget/split-container.md) |
| TabControl | done | `TabControl`, `Sidebar` |
| TableLayoutPanel | done | `Grid`, `Fields` |
| TextBox | done | `TextBox`, `MultilineTextBox` |
| ToolStrip | done | `ToolStrip` |
| ToolStripContainer | dropped (Phase 2) | docking toolbars to the four edges is the docking subsystem (below); a `Form` has one tool strip and a status strip |
| TrackBar | done | [TrackBar](widget/trackbar.md) |
| TreeView | done | [TreeView](widget/treeview.md) |
| WebBrowser | done | [WebBrowser](widget/webbrowser.md) |
| HScrollBar, VScrollBar | dropped | a bare scroll bar scrolls nothing of its own here; `ScrollPanel` and the scrolling controls bring theirs, and `TrackBar` is the value slider |
| Splitter (the old control) | dropped | replaced by `SplitContainer` in WinForms itself |
| DataGrid (the old control) | dropped | replaced by `DataGridView` |
| BindingNavigator | dropped | tied to `BindingSource`; `Bind` and `DataGridView`'s paging cover the cases |
| Chart | done | [Chart](widget/chart.md) (a Windows Forms Chart control exists in .NET Framework 4) |
| ElementHost, WPF interop | dropped | no WPF |
| UserControl | done | `Custom`: [Controls of your own](widget.md#controls-of-your-own) |
| MDI parent and children | dropped | WinForms itself steers away from MDI; `TabControl` and separate `Form`s cover it |

## Components

| WinForms | Status | Where |
|---|---|---|
| ColorDialog | done | [ColorDialog](widget/colordialog.md) |
| ContextMenuStrip | done | `ContextMenu`, `MenuButton`: [widget.md](widget.md#menus) |
| ErrorProvider | done | [ErrorProvider](widget/errorprovider.md) |
| FolderBrowserDialog | done | `SelectFolder` |
| FontDialog | done | [FontDialog](widget/fontdialog.md) |
| HelpProvider | dropped | `SetTooltip` and the platform's help are what apps use; no help-file formats |
| ImageList | done | [ImageList](widget/imagelist.md) |
| MenuStrip | done | `Form.MenuStrip`, `SetMenuBar`: [Menus](../menus.md) |
| MessageBox | done | `Alert`, `Confirm`, `InputString`, `InputSelect` |
| NotifyIcon | done | `NotifyIcon` |
| OpenFileDialog, SaveFileDialog | done | `OpenFile`, `OpenFiles`, `SaveFile` |
| PageSetupDialog, PrintDialog, PrintDocument, PrintPreviewControl, PrintPreviewDialog | dropped (Phase 2) | fhtool doesn't print; native print APIs would be engine work (Track 3) if an app needs them |
| StatusStrip | done | `StatusStrip` |
| Timer | done | [Timer](widget/timer.md) |
| ToolTip | done | `SetTooltip` on any control |
| BackgroundWorker | dropped | goroutines, `Post` and `Invoke` do it: [Threads](widget.md#threads) |
| BindingSource | dropped | `Bind` and `RowSource` |
| Drag and drop | done | [Drag and drop](widget/drag-and-drop.md) |
| Shortcuts, `AcceptButton`, `CancelButton` | done | [Form keys](widget/form-keys.md) |
| Validation (`Validating`, `ValidateChildren`) | done | [ErrorProvider](widget/errorprovider.md) |
| Mnemonics | done | [Mnemonics](widget/mnemonics.md) (a `Label` focuses the next control with `SetUseMnemonic(true)`; off by default, unlike WinForms) |

## Pieces with a design of their own

- **Docking** (dockable panes, `ToolStripContainer`): dropped for Phase 2. It is
  a subsystem of its own; `SplitContainer` and `TabControl` cover fixed
  layouts. Revisit on request.
- **Printing** (`PrintDocument`, preview, page setup): dropped for Phase 2.
  fhtool doesn't print; native print APIs would be engine work (Track 3) if an
  app needs them.

## Open rows

Every Phase 2 row is done, or dropped with the reason above. What is left of
ListView is its item groups (`ListViewGroup`): the icon views need
`GridState.Header` in `ui`, which is there, and the widget takes it next.

Every row marked done has a page or a section of [widget.md](widget.md), unit
tests in `ui/widget`, and an entry in the [widget tour](../../examples/widget-tour)
(`go run ./examples/widget-tour`) with a test in the tour's package: the
Components tab holds the ImageList, a control of your own, mnemonics and a
LinkLabel.
