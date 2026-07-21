# 05: cmd/goimportalias 骨格・モード分岐・vet 統合テスト

## 目的

単一バイナリ `goimportalias` を作成し、起動時の引数で「vet ツールモード（unitchecker）」と「CLI モード」に自動分岐する骨格を用意する。まず vet ツールモードを動く状態にし、`go vet -vettool=` 経由の実挙動を統合テストで担保する。CLI モードは後続 issue のためのスタブでよい。

## 対象 DEC・FR

- DEC-1.4（起動引数で 2 モードに自動分岐。vet モードは `unitchecker.Main(importalias.Analyzer)` に委譲）
- DEC-1.4.1 / FR-6.15（vet モードはファイルシステムへ一切書き込まない）
- DEC-6.1（`golang.org/x/tools/go/analysis/unitchecker` を採用）
- FR-6.2 / FR-6.3（`go vet -vettool=` 実行、golangci-lint 連携は Analyzer export により自明）

## 変更ファイル

- `cmd/goimportalias/main.go`（新規。`os.Args` を覗いて vet ツール呼び出し（`-V=full` ハンドシェイク／`*.cfg` を唯一の引数に取る規約）を判定 → `unitchecker.Main`。それ以外は `runCLI(os.Args[1:])` を呼ぶ（当面は「未実装」と表示して終了する薄いスタブ））
- `cmd/goimportalias/main_test.go`（新規。vet 統合テスト）

## 実装メモ

- `unitchecker.Main` は内部で `os.Args` をグローバルに読む（`docs/02notice.md` 第2回で確認済み）。**呼ぶ前に `flag.Parse()` してはいけない**（二重パース）。
- モード判定は `docs/draft.md` の `looksLikeVetToolInvocation` スケッチを起点にし、実挙動を見て固める（DEC-1.4 が「実装時に固める」としている部分）。
- 統合テストは `go test` の中で本バイナリを `go build -o <tmp>` し、`go vet -vettool=<tmp> ./...`（既知の不整合を持つ testmodule に対して）を exec して、非ゼロ終了かつ診断が stderr に出ることを確認する方式が素直。testmodule は `testdata/` 配下に別モジュールとして用意してもよい。

## 終了条件

- [ ] `go build ./...` が成功し、`cmd/goimportalias` のバイナリが生成できる。
- [ ] vet 統合テスト green: 実バイナリを build → 既知の不整合パッケージに対し `go vet -vettool=` 経由で実行し、(a) 非ゼロ終了 (b) 期待する診断メッセージが出力される。
- [ ] 正常（不整合なし）パッケージでは終了コード 0。
- [ ] vet モードでファイルが書き換わらない／設定ファイルが生成されないことを確認（FR-6.15）。
- [ ] `go test ./...` 全 green。

## 依存

推奨: 02/03/04 のいずれかで analyzer の診断が拡張された後だと統合テストの確認対象が増える。最低限、現状の FR-6.10 診断だけでも本 issue は成立する。
