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
| Label | done | `Label` (`&` mnemonics: [Mnemonics](widget/mnemonics.md)) |
| LinkLabel | done | `LinkLabel` |
| ListBox | done | [ListBox](widget/listbox.md) |
| ListView | **pending** | [#132](https://github.com/lang315/mygo/pull/132), in review |
| MaskedTextBox | done | [MaskedTextBox](widget/maskedtextbox.md) |
| MonthCalendar | done | [DateTimePicker](widget/date-time-picker.md) |
| NumericUpDown | done | `NumericUpDown` |
| Panel | done | `VBox`, `HBox`, `Card`, `ScrollPanel` (AutoScroll) |
| PictureBox | done | `PictureBox` |
| ProgressBar | done | `ProgressBar` |
| PropertyGrid | **pending** | held for the owner's call (likely later or dropped for fhtool) |
| RadioButton | done | `RadioButton`, `RadioGroup` |
| RichTextBox | done | [RichTextBox](widget/richtextbox.md) |
| SplitContainer | done | [SplitContainer](widget/split-container.md) |
| TabControl | done | `TabControl`, `Sidebar` |
| TableLayoutPanel | done | `Grid`, `Fields` |
| TextBox | done | `TextBox`, `MultilineTextBox` |
| ToolStrip | done | `ToolStrip` |
| ToolStripContainer | dropped | docking toolbars to the four edges; a `Form` has one tool strip and a status strip, and docking is a larger piece of its own (below) |
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
| MDI parent and children | **pending** | document tabs stand in; a design of its own (Docking, below) |

## Components

| WinForms | Status | Where |
|---|---|---|
| ColorDialog | done | [ColorDialog](widget/colordialog.md) |
| ContextMenuStrip | done | `ContextMenu`, `MenuButton`: [widget.md](widget.md#menus) |
| ErrorProvider | done | [ErrorProvider](widget/errorprovider.md) |
| FolderBrowserDialog | done | `SelectFolder` |
| FontDialog | done | [FontDialog](widget/fontdialog.md) |
| HelpProvider | dropped | `SetTooltip` and the platform's help are what apps use; no help-file formats |
| ImageList | **pending** | `widget.ImageList`, next |
| MenuStrip | done | `Form.MenuStrip`, `SetMenuBar`: [Menus](../menus.md) |
| MessageBox | done | `Alert`, `Confirm`, `InputString`, `InputSelect` |
| NotifyIcon | done | `NotifyIcon` |
| OpenFileDialog, SaveFileDialog | done | `OpenFile`, `OpenFiles`, `SaveFile` |
| PageSetupDialog, PrintDialog, PrintDocument, PrintPreviewControl, PrintPreviewDialog | **pending** | Printing, a design of its own |
| StatusStrip | done | `StatusStrip` |
| Timer | done | [Timer](widget/timer.md) |
| ToolTip | done | `SetTooltip` on any control |
| BackgroundWorker | dropped | goroutines, `Post` and `Invoke` do it: [Threads](widget.md#threads) |
| BindingSource | dropped | `Bind` and `RowSource` |
| Drag and drop | done | [Drag and drop](widget/drag-and-drop.md) |
| Shortcuts, `AcceptButton`, `CancelButton` | done | [Form keys](widget/form-keys.md) |
| Validation (`Validating`, `ValidateChildren`) | done | [ErrorProvider](widget/errorprovider.md) |
| Mnemonics | done | [Mnemonics](widget/mnemonics.md) (a `Label` that focuses the next control: pending, a small follow-up) |

## Pieces with a design of their own

- **Docking** (MDI, `ToolStripContainer`, dockable panes): pending, not
  started.
- **Printing** (`PrintDocument`, preview, page setup): pending, not started.

## Open rows

ListView (#132), PropertyGrid, ImageList, Label mnemonics, MDI/Docking and
Printing. Everything else is done or dropped with the reason above.
