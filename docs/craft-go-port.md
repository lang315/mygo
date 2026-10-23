# Port bộ Craft (storytold) từ Rust sang Go — bản tổng hợp

_Cập nhật: 2026-10-14 · Phạm vi: chỉ app desktop (Windows, macOS, Linux), bỏ qua web UI · Hướng đi: xây trên **mygo**._

## 1. Tóm tắt

- Tổ chức **storytold** (nhóm ArtCraft) có ba nhóm repo:
  - **Craft App Suite**: 14 app viết lại các phần mềm quen thuộc (Photoshop, Excel, Word…);
  - **ArtCraft**: sản phẩm AI chính, dùng Tauri + React;
  - các repo cũ: plugin Unreal, ML, hạ tầng.
- Cả 14 app Craft dùng **chung một stack UI**: Rust + **egui/eframe 0.36** + **wgpu 30** + **winit 0.30** + **AccessKit**.
- Thứ đáng port sang Go nhất là **kiến trúc**, không phải egui:
  - engine tách khỏi UI;
  - command bus `run(id, json)` dùng chung cho UI, CLI và AI agent;
  - thư viện widget dùng chung (`craft-ui`).
- **Không tự viết lõi UI nữa, mà xây trên [mygo](https://github.com/egoist/mygo)** (fork: [lang315/mygo](https://github.com/lang315/mygo)). mygo đã có sẵn những gì cần thiết:
  - lõi UI tự vẽ trên GPU (Metal, D3D11, OpenGL);
  - lớp platform viết bằng Go thuần, không cgo, gọi native qua purego và `syscall`;
  - IME, accessibility, menu, hộp thoại, đóng gói app.
- Lớp riêng của Craft (docking, ribbon, command bus, control server, MCP, các thư viện kiểu craft-libs) đặt ở **repo riêng `craft-go`**, phụ thuộc vào mygo, **không viết trong bản fork**.

## 2. Các repo trong storytold

> Danh sách repo trên trang GitHub của tổ chức hiển thị thiếu. Một số tên cũ chỉ chuyển hướng sang repo mới:
> sheetcraft → gridcraft, slidecraft → deckcraft, drawcraft → vectorcraft, printcraft → pdfcraft.

### 2.1 Craft App Suite (Rust, egui)

| Repo | Viết lại của | Ghi chú |
|---|---|---|
| photocraft | Photoshop | Bút tablet, PSD, RAW, HEIF |
| vectorcraft | Illustrator | SVG, EPS, Affinity, pathops |
| designcraft | InDesign | IDML, EPUB, PDF |
| figcraft | Figma | Có server collab riêng; không dùng craft-ui |
| wordcraft | Word | DOCX, kiểm tra chính tả, Zotero |
| gridcraft | Excel | Công thức, xlsx, pivot, chart |
| deckcraft | PowerPoint | Tự viết codec H.264/HEVC/AV1/VP9; không dùng craft-ui |
| pdfcraft | Acrobat | Bật cả backend `glow` lẫn `wgpu` |
| soundcraft | Pro Tools | Host plugin VST3, CLAP, AU |
| cadcraft | AutoCAD | DWG, DXF, constraints |
| filmcraft | Premiere Pro | Codec, audio DSP, TTS |
| effectcraft | After Effects | Lottie, PSD, hiệu ứng GPU |
| lightcraft | Lightroom | Catalog, RAW, denoise; bật cả `glow` lẫn `wgpu` |
| craft-launcher | Trình cài đặt và cập nhật cả bộ app | |
| **craft-libs** | Thư viện dùng chung | Xem mục 3.3 |

### 2.2 ArtCraft (ngoại lệ, dùng web UI)

- **artcraft**:
  - app desktop dùng **Tauri 2.11**;
  - frontend **React 18 + Vite + Tailwind + Radix UI + Zustand**, quản lý bằng monorepo Nx;
  - canvas 2D dùng **Konva**, 3D dùng **three.js**;
  - có thêm `artcraft-native`, bản thử viết lại bằng **egui** (không Tauri, không webview).
- **artcraftx**: bản rút gọn, cũng là Tauri + React.
- **artcraft-services**: backend **actix-web + Tokio + Redis**, kèm vài web frontend (React/Remix, Next.js).

### 2.3 Các repo cũ (không liên quan UI desktop)

- **ML / giọng nói (Python)**: storyteller-ml, vits-finetuning, realtime-voice-conversion, FineTrainers-Conditioning, cloud-worker.
- **Unreal / Unity, motion capture, depth sensor**: EventCenterPlugin, ActorSpawningPlugin, RuntimeAudioImporter, RuntimeFilesDownloader, UE4-OSC, UnrealZeroMQ, LiveScan3D, Azure-Kinect-Sample-for-Unity, xsens-packet-send, k4a-sys-temp, point-generator, bevy-mocap, storyteller-bevy.
- **Khác**: spark (3D Gaussian Splatting cho three.js), irsa-manager (Go), terraform-aws-observability-accelerator, github-media, html_test, placeholder-artcraft (đã archive).

## 3. Stack và kiến trúc của bộ Craft (bản Rust)

### 3.1 Stack chung (đọc từ `Cargo.lock`)

| Vai trò | Thư viện | Phiên bản |
|---|---|---|
| Lõi GUI immediate-mode | egui | 0.36.2 |
| Khung app (cửa sổ, vòng lặp) | eframe | 0.36.2 |
| Render GPU | wgpu (qua egui-wgpu) | 30.0.1 |
| Cửa sổ, input | winit | 0.30.13 |
| Accessibility | AccessKit | 0.24.1 |
| Menu native macOS | muda | 0.19–0.21 |
| Hộp thoại file | rfd | 0.16–0.17 |
| Hình học, font | kurbo, skrifa | |
| Test UI | egui_kittest (snapshot) | 0.36 |

Tích hợp riêng từng hệ điều hành (ví dụ ở PhotoCraft, GridCraft):

- **macOS**: `objc2` / `objc2-app-kit`; `fmv-macos-events` để nhận sự kiện mở file từ Finder.
- **Windows**: `winsafe` cho title bar tùy biến và vị trí con trỏ.
- **Linux**: `x11rb`, `zbus` (XDG portal), `wl-clipboard-rs`.

### 3.2 Cấu trúc một app (mẫu GridCraft)

```
crates/
  core, model, formula, calc, ...   # engine thuần, KHÔNG phụ thuộc egui
  engine/                           # Session + command bus
    cmd/ edit.rs format.rs file.rs pivot.rs print.rs ...   (hơn 20 nhóm lệnh)
  ui-egui/                          # UI mỏng (~20k dòng): grid.rs, ribbon.rs, dialogs.rs ...
  mcp/                              # MCP server (JSON-RPC 2.0 tự viết)
apps/
  gridcraft/         # desktop: main.rs, native_menu.rs, control_server.rs, print.rs ...
  gridcraft-cli/
  gridcraft-web/     # (bỏ qua trong bản port)
```

Các ý tưởng cốt lõi:

1. **Engine độc lập UI.** Crate UI tự mô tả là "swappable — nothing below depends on it".
2. **Command bus.** Mọi thao tác đi qua `Session::run(id: &str, params: Json) -> Result<Json, String>`. Chiều ngược lại, engine gửi yêu cầu cho UI qua `take_ui_requests()`.
3. **Control server.** TCP server trên localhost nhận JSON-lines `{"id","method","params"}` và trả lời `{"id","ok",...}`. Binary MCP nối vào server này để AI agent điều khiển app.
4. **UI tự vẽ.** Lưới ô, ribbon, keytips, backstage đều vẽ bằng painter của egui, không dùng widget bảng có sẵn.
5. **Chart độc lập toolkit.** `gridcraft-chart` tính layout ra các primitive, sau đó UI mới vẽ.

### 3.3 craft-libs (thư viện dùng chung)

| Crate | Chức năng |
|---|---|
| craft-ui (~5k dòng) | Control egui dùng lại được: buttons, **docking** (phần lớn nhất), tabs, layout (toolbar, resize), theme, icons, printing |
| craft-print | In qua CUPS và Windows |
| craft-cms | Quản lý màu ICC v2/v4, viết bằng Rust thuần |
| craft-settings | Lưu cấu hình: thư mục theo nền tảng, config.toml, khôi phục và migrate |
| craft-atomic | Ghi và thay file an toàn khi crash |
| craft-quit | Thoát app đúng chuẩn từng hệ điều hành |

Nguyên tắc của craft-ui: app tự quản lý asset, theme, layout, lệnh và ngữ nghĩa tài liệu. Component chỉ nhận style và trả về response; không cần engine.

## 4. Vai trò của purego

- **purego** (ebitengine/purego) cho phép Go **gọi hàm C mà không cần cgo** (dlopen/dlsym). Đây **không phải** thư viện UI.
- Các nền tảng Tier 1: Windows, macOS, Linux, Android, iOS trên amd64 và arm64. Giới hạn chủ yếu ở các kiến trúc Tier 2.
- **Không cần fork.** mygo đã dùng purego v0.11.1 cho macOS và Linux, và `syscall` cho Windows.

## 5. Đánh giá mygo làm nền tảng

> Chưa chạy được test: repo yêu cầu Go 1.27.1, môi trường đánh giá chỉ có Go 1.24. Đánh giá dựa trên đọc code và thử merge bằng git.

### 5.1 mygo là gì

- Framework app desktop viết bằng Go. Mỗi cửa sổ hiển thị một trong hai loại giao diện:
  - trang web trong webview của hệ điều hành;
  - **UI native** viết bằng package `ui` và tự vẽ trên GPU.
- Khi chỉ làm desktop với UI native, có thể bỏ qua hoàn toàn phần webview và WASM.
- **Go thuần, không cgo**: build được cho mọi nền tảng từ bất kỳ máy nào.
- Quy mô khoảng 234k dòng Go; riêng `ui` có khoảng 55k dòng code và 24k dòng test. License MIT.
- Phụ thuộc: `purego` v0.11.1, `go-text/typesetting` v0.3.5, `golang.org/x/image`. Yêu cầu Go 1.27.1.

Ví dụ một view (hàm Go chạy lại mỗi frame, gần với egui):

```go
func (s *counter) view(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(12).Children(func() {
		ui.Text(c, fmt.Sprint(s.n)).FontSize(40).Bold()
		if ui.PrimaryButton(c, "Increment").Clicked() {
			s.n++
		}
	})
}
```

### 5.2 So với stack Craft

| Thành phần trong Craft (Rust) | Trong mygo | Tình trạng |
|---|---|---|
| egui (lõi UI) | Package `ui`, có thêm layout flexbox và grid | Có |
| epaint + wgpu | `internal/gpu`: **Metal** (macOS), **D3D11** (Windows), **OpenGL** (Linux); có thêm `internal/raster` | Có (chỉ để vẽ UI) |
| winit + eframe | `internal/{darwin,windows,linux}` | Có |
| Text và IME | `internal/text`, IME riêng từng hệ điều hành (macOS dùng `NSTextInputClient`, Linux có preedit) | Có, cần thử với Telex |
| AccessKit | NSAccessibility (macOS), UI Automation (Windows), hỗ trợ trên Linux | Có |
| muda / rfd / arboard | Menu native, hộp thoại file, clipboard, tray, phím tắt toàn cục, deep link | Có |
| egui_kittest | View chạy được không cần cửa sổ (`headless`), có inspector | Có |
| craft-ui | `Split`, `Tabs`, `toolbar`, `sidebar`, `tree`, `outline`, `table` / `tableview`, `gridview`, `combobox`, `colorpicker`, `editor`, `chart`, `canvas`, `toast`, `tooltip` | Một phần: thiếu **docking**, **ribbon** |
| craft-settings | `window_state` (lưu trạng thái cửa sổ) | Một phần |
| Đóng gói app | dmg, cài đặt NSIS, gói deb, ký số, notarize, auto-update có delta | Có (Craft không có sẵn) |
| Command bus, control server, MCP | Không có | Tự viết |
| craft-print, craft-cms, craft-atomic | Không có | Tự viết |
| GPU compute cho xử lý ảnh | Không có | Tự viết khi cần |

### 5.3 Bản fork lang315/mygo

- Tách ra từ upstream ở bản **0.4.8** (commit `c0e1068`, ngày 2026-10-13).
- Có 5 commit riêng, khoảng 4.600 dòng (trong đó khoảng 1.800 dòng test):
  - bảng có checkbox, menu theo dòng, copy dữ liệu, sự kiện riêng;
  - `TableView` xử lý 100.000 dòng;
  - lọc trên bảng đã sắp xếp mà không phải sắp xếp lại;
  - ví dụ `examples/account-manager`;
  - khôi phục cách đo thời gian frame.
- Đây là nền tốt cho một lưới dữ liệu kiểu GridCraft.

### 5.4 Rủi ro

- **Upstream đổi rất nhanh.** Chỉ khoảng một ngày sau khi fork, upstream đã có thêm **43 commit**. Thử merge upstream vào fork thì **5 file bị conflict**: `ui/list.go`, `ui/paint.go`, `ui/api_gen.go`, `ui/table_test.go`, `examples/gallery/main_test.go`.
- **Trùng việc với upstream.** Upstream đang làm cùng mảng bảng: `HeaderMenu` (#308), sắp xếp nhiều cột `SortBy` (#312).
- **API chưa ổn định.** Phiên bản 0.4.x; repo có công cụ `uimigrate` / `uivet` để chuyển code sang API mới.
- **Phụ thuộc một người.** Phần lớn code do một maintainer viết (khoảng 200 commit).
- **Renderer chỉ để vẽ UI.** Chưa có GPU compute tổng quát, nên app nặng đồ họa (PhotoCraft, LightCraft, FilmCraft) cần đường riêng cho xử lý ảnh.

## 6. Chiến lược làm việc với fork

1. **Merge upstream ngay**, khi conflict còn ít; sau đó merge định kỳ (lý tưởng là hằng ngày khi upstream còn đổi nhanh).
2. **Gửi phần cải tiến bảng thành PR lên egoist/mygo.** Nếu được nhận, bạn không phải tự giữ phần đó nữa.
3. **Giữ fork càng gần upstream càng tốt.** Chỉ dùng fork cho các bản sửa đang chờ upstream nhận. Giữ nguyên module path `github.com/egoist/mygo` để merge dễ.
4. **Không viết lớp Craft trong fork.** Đặt nó ở repo riêng và trỏ tới fork bằng `replace`:

```go
// craft-go/go.mod
module github.com/lang315/craft-go

go 1.27.1

require github.com/egoist/mygo v0.4.8

// Chỉ khi cần bản sửa chưa có ở upstream:
replace github.com/egoist/mygo => github.com/lang315/mygo v0.0.0-<commit>
```

## 7. Khung repo đề xuất

**Repo `craft-go`** (thay cho craft-libs, xây trên mygo):

```
craft-go/
  craftui/
    dock/        # docking: panel kéo thả, ghép tab (port từ craft-ui/docking)
    ribbon/      # ribbon kiểu Office + keytips
    grid/        # lưới ô ảo hóa kiểu GridCraft (dựa trên TableView/gridview)
    theme/       # theme chung của bộ Craft trên ui.Theme của mygo
    icons/
  craftlib/
    command/     # command bus: Registry, Run(id, params) -> result
    control/     # control server JSON-lines trên localhost
    mcp/         # MCP server (JSON-RPC 2.0) nối tới control server
    settings/    # cấu hình, migrate (bổ sung window_state của mygo)
    atomic/      # ghi file an toàn khi crash
    print/       # in qua CUPS / Windows
    cms/         # quản lý màu ICC (khi cần)
```

**Mỗi app** (ví dụ `gridcraft-go`):

```
gridcraft-go/
  engine/               # model + lệnh, KHÔNG import mygo hay craftui
    session.go          # Run(id string, params json.RawMessage) (json.RawMessage, error)
    cmd/                # edit.go, format.go, file.go ...
  ui/                   # màn hình của app: mygo/ui + craftui
  cmd/gridcraft/        # mygo.App, cửa sổ, menu, khởi động control server
  cmd/gridcraft-cli/    # chạy lệnh trên file, không cần UI
  cmd/gridcraft-mcp/    # MCP server
```

Command bus gợi ý:

```go
package command

type Handler func(ctx context.Context, params json.RawMessage) (json.RawMessage, error)

type Registry struct{ handlers map[string]Handler }

func (r *Registry) Register(id string, h Handler)
func (r *Registry) Run(ctx context.Context, id string, params json.RawMessage) (json.RawMessage, error)

// Engine -> UI: yêu cầu mở dialog, chọn ô, cuộn tới... (tương đương take_ui_requests)
type UIRequest struct {
	Kind   string
	Params json.RawMessage
}
```

UI, CLI, control server và MCP đều gọi cùng `Registry.Run`. Theo luật luồng của mygo, callback và handler chạy trên main thread. Khi lệnh đến từ goroutine của control server, chạy nó qua `mygo.RunOnMain(func(){ ... })`, rồi yêu cầu vẽ lại view (`Invalidate`).

## 8. Lộ trình (đã cập nhật)

1. **Đồng bộ fork:** merge upstream, sửa 5 conflict, gửi PR phần bảng.
2. **Kiểm tra sớm trên cả 3 hệ điều hành:** gõ tiếng Việt qua IME (Telex với Unikey/EVKey, bộ gõ mặc định của macOS, ibus/fcitx trên Linux), chọn và sửa text, cuộn bảng 100k dòng.
3. **`craftlib/command` + `control` + `mcp`:** viết phần kiến trúc trước, vì nó không phụ thuộc UI.
4. **App mẫu trọn vòng:** GridCraft bản rút gọn, gồm engine, command bus, lưới dựa trên `TableView`, menu, CLI, control server.
5. **`craftui/dock` và `craftui/ribbon`:** port từ craft-ui (docking khoảng 1.700 dòng chưa tính test) và `ribbon.rs` của GridCraft (khoảng 2.500 dòng).
6. **Các app khác:** ưu tiên app dạng tài liệu (WordCraft, DeckCraft, PdfCraft). App nặng đồ họa để sau, khi đã có đường xử lý ảnh trên GPU.

Không còn cần các bước "tự viết lõi `ui` / `ui/paint`", "backend Ebitengine", "backend purego + wgpu-native" của bản kế hoạch trước, vì mygo đã có.

## 9. Rủi ro và điểm khó còn lại

- **Đồng bộ với upstream:** chi phí lớn nhất nếu fork lệch nhiều (mục 5.4).
- **IME tiếng Việt:** mygo có hỗ trợ IME, nhưng cần thử thực tế với Telex trên từng hệ điều hành.
- **Docking và ribbon:** phải tự viết; docking là phần phức tạp nhất của craft-ui.
- **Xử lý ảnh nặng:** Go có garbage collector, SIMD hạn chế, và mygo không có GPU compute. Cần đường riêng cho PhotoCraft, LightCraft, FilmCraft.
- **API mygo còn đổi:** khóa phiên bản trong `go.mod`, và dùng `uimigrate` / `uivet` khi nâng cấp.

## 10. License

- Các app Craft và craft-libs: `MIT OR Apache-2.0` (có đủ `LICENSE-MIT` và `LICENSE-APACHE`).
- egui: MIT/Apache-2.0. purego: Apache-2.0. mygo: MIT.
- Được port ý tưởng hoặc code, miễn giữ LICENSE/NOTICE và ghi nguồn.
- Riêng ArtCraft dùng license khác, cần đọc kỹ trước khi lấy code.

## 11. Nguồn

- https://github.com/orgs/storytold/repositories
- https://getartcraft.com/apps
- https://github.com/storytold/gridcraft
- https://github.com/storytold/photocraft
- https://github.com/storytold/craft-libs
- https://github.com/storytold/artcraft
- https://github.com/egoist/mygo
- https://github.com/lang315/mygo
- https://github.com/ebitengine/purego
