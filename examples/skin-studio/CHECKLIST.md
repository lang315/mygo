# Skin P1 checklist

One row per control of section 5 of the skins spec ("Controls": the widgets
`ui/widget` has). Part says what the skin gives it: **metric** (a size),
**state** (hover, pressed, selected, checked, focus, disabled) or **literal**
(every color it paints comes from the tokens). "Shown by" names the studio
page that shows the control (`go run ./examples/skin-studio`, not run by tests)
and the test that holds it. A row without a page or a test is not ticked.

Tests of every row: `TestStudioEveryPage` (the page builds in None, in Calm
× 5 palettes, Classic × 2 and HighContrast, × light and dark, and Classic at
text scale 1.5, no caption clipped), `TestStudioGridPage` (rows 30, header 30,
tool strip controls 36), `TestSentinelSkinReachesEveryPixel`
(`ui/widget/skin`: under a sentinel skin of each built-in skin's metrics and
looks every painted pixel of the gallery,
which holds each control below except the message boxes' dialogs, is a sentinel
color) and `TestNoNewLiteralColors` (`ui/widget`: no `ui.Hex`/`ui.RGB` in a
skinned paint path). The studio's Edit tab (S6) has its own: `TestEditor*`
(`editor_test.go`: the model, its file round trip, every `Tokens` leaf) and
`TestEdit*` (`editpanel_test.go`: an edit reaches a control, Check's report
in the status strip, a bad value refused at its field, the appearance shown
apart from the app's, Open and Save, the tool strip's "Edited");
`TestShotsMatchMain` compares the shots with main's outside the Edit tab's
header. Rows below add their own.

| | Control | Part | Shown by |
|---|---|---|---|
| [x] | Button, primary, destructive, disabled | metric 36, state (hover, pressed, disabled), button outline | page Buttons and toggles; `TestCalmSizesWidgets`, `TestButtonBorder` (`ui`), `TestWidgetSetSizeWinsOverMetrics` |
| [x] | DropDownButton, MenuButton, LinkLabel | metric 36, state | page Buttons and toggles; `TestSentinelSkinReachesEveryPixel` |
| [x] | Toggle, ToolButton | metric 36, state (checked) | page Buttons and toggles (Toggle), tool strip (ToolButton in the gallery); `TestSkinToolStrip` |
| [x] | CheckBox, CheckBoxGroup | metric 15, state (checked, mixed) | page Buttons and toggles; `TestCalmSizesWidgets` |
| [x] | RadioButton, RadioGroup | metric (dot), state (checked) | page Buttons and toggles; `TestMetricsSizeControls` (`ui`) |
| [x] | Switch | metric 40 × 22, state (on, off) | page Buttons and toggles; `TestMetricsSizeControls` (`ui`) |
| [x] | TextBox, MaskedTextBox, MultilineTextBox | metric 36, state (field outline, focus, disabled) | page Text and choices; `TestCalmSizesWidgets` |
| [x] | ComboBox, editable ComboBox | metric 36, state, literal (drop-down) | page Text and choices, tool strip; `TestCalmSizesWidgets`, `TestStudioGridPage`, `TestStudioSwitches` |
| [x] | TagSelect | metric 36, state | page Text and choices; `TestSentinelSkinReachesEveryPixel` |
| [x] | ListBox, CheckedListBox | metric row 30, state (hover, selected) | page Lists and trees; `TestSkinListBoxRows` |
| [x] | NumericUpDown, DomainUpDown, RangeSpin | metric 36, state | page Numbers and dates; `TestCalmSizesWidgets`, `TestSkinDomainUpDown` |
| [x] | DateTimePicker, MonthCalendar | metric 36, state (chosen day keeps the accent) | page Numbers and dates; `TestCalmSizesWidgets`, `TestDateTimePickerSetSizeWinsOverMetrics` |
| [x] | TrackBar | state (thumb, track) | page Numbers and dates; `TestSentinelSkinReachesEveryPixel` |
| [x] | Fields, SettingsRows | metric (rows), literal (lines), look (SettingsCard: a card per section, else Card's), description (`Describe`) | page Text and choices, on Chrome; `TestSentinelSkinReachesEveryPixel`, `TestSettingsCardsFramedByTheCardLook`, `TestSettingsCardsUnderCalm` |
| [x] | HBox, VBox, ButtonBox | none: layout only, the skin never moves controls | every page; `TestStudioEveryPage` |
| [x] | Grid (layout), FlowPanel | none: layout only | page Containers; `TestStudioEveryPage` |
| [x] | GroupBox | literal (outline, title) | page Containers; `TestSentinelSkinReachesEveryPixel` |
| [x] | Card | state (elevation, chrome vs content), literal | page Containers; `TestSkinCard` |
| [x] | ScrollPanel | literal (scroll bar) | page Containers; `TestSentinelSkinReachesEveryPixel` |
| [x] | SplitContainer | literal (splitter) | page Containers; `TestSentinelSkinReachesEveryPixel` |
| [x] | DataGridView | metric row 30, header 30, state (hover, selected), status column | page Grid; `TestStudioGridPage`, `TestSetRowHeightWinsOverSkin`, `TestKeptRowsResizeOnUse` |
| [x] | ListView, details and icons | metric row 30, state (hover, selected) | page Lists and trees; `TestSkinListViewRows`, `TestSkinListViewRowsWithImages` |
| [x] | TreeView | metric row 30, state (hover, selected) | page Lists and trees; `TestCalmSizesWidgets` |
| [x] | TabControl | metric strip 40, state (open tab line), literal (strip on chrome) | the studio's own tabs and page Containers; `TestSkinTabStrip` |
| [x] | Sidebar | state (chosen item) | page Containers; `TestSentinelSkinReachesEveryPixel` |
| [x] | MenuBar | metric 28 in 34, state (hover) | page Containers; `TestSkinMenuBar`, `TestSkinMenuBarFollowsTextScale` |
| [x] | MenuStrip | none: the OS draws it, out of P1 by design (plan, Changes from the spec 3) | the MenuBar row |
| [x] | ToolStrip | metric 36, state (hover, pressed), chrome | the studio's tool strip; `TestStudioGridPage`, `TestSkinToolStrip`, `TestSkinToolStripFollowsTextScale`, `TestCheckControlStates` (`ui/widget/skin`) |
| [x] | StatusStrip | chrome, literal | the studio's status strip; `TestSkinStatusStrip` |
| [x] | HeaderBar | chrome, literal | the studio's header bar; `TestSkinHeaderBar` |
| [x] | Dialog, message boxes (Alert, Confirm, Input) | literal (kind icons), content layer | page Dialogs and messages; `TestMessageIconsUseSkinColors` |
| [x] | ProgressBar, Spinner | state, literal | page Chart and progress, the status strip; `TestSentinelSkinReachesEveryPixel` |
| [x] | ErrorProvider | literal (error glyph) | page Text and choices; `TestErrorGlyphUsesDangerOn` |
| [x] | Chart (line, bar, pie) | literal (series colors) | page Chart and progress; `TestChartUsesSkinColors` |
| [x] | Badge | metric 20 (text scale), literal (status families' Subtle, Text, Base), look `Badge` | page Dialogs and messages; `TestStatusPage`, `TestBadgeCalmSuccess`, `TestBadgeRunningDetail`, `TestBadgeLook`, `TestBadgeSkinSwitch` (`ui/widget`) |
| [x] | InfoBar | literal (Subtle, Border, Base icon), look `InfoBar` | page Dialogs and messages; `TestStatusPage`, `TestInfoBarWarning`, `TestInfoBarLook`, `TestInfoBarCloseOnlyByUser` (`ui/widget`) |
| [x] | EmptyState | literal (TextMuted, Title role, Suggested action), look `EmptyState` | page Dialogs and messages; `TestStatusPage`, `TestEmptyState`, `TestEmptyStateLook` (`ui/widget`) |
