# 04: analyzer の config 読込・ignore・strict flag

## 目的

go vet analyzer（D1）が `importalias.json` を read-only で読み込み、`ignore` パターンによるスキップと設定ファイルの alias 指定（多数決より優先）を反映できるようにする。あわせて `-strict` フラグを提供する。現状 analyzer は `decide.Decide(occs, nil, ...)` と config を無視している。

## 対象 DEC・FR

- DEC-11.2（`pass.Module` 経由でモジュールルートを推定し `importalias.json` を `shape.Load`。読めなければ stderr にログ＋「設定なし」で継続。read-only なので FR-6.15 に抵触しない）
- FR-5.10〜5.12（`ignore` の記法・除外・優先順位不要）
- DEC-3.2（`strict` は設定ファイルではなく実行時フラグ。vet モードは `-importalias.strict` bool 既定 false）
- FR-4.1〜4.3（config 優先順位。ロジック本体は `shape.File.Lookup` に既にある）

## 変更ファイル

- `analyzer.go`（`pass.Module` から `filepath.Join(pass.Module.Dir, "importalias.json")` を `shape.Load`。`pass.Module == nil` / ロード失敗時は stderr ログ＋ `cfg = nil` で継続。`ignore` マッチする package はスキップ。`decide.Decide(occs, cfg, decide.Options{Strict: strictFlag})`。`Analyzer.Flags` に `strict` を追加）
- 必要なら `internal/shape/config.go`（`ignore` パターンマッチ用のヘルパー。`Lookup` と同じ prefix/`foo/...` 判定を再利用できる形で。DEC-11.15 に倣い判定は shape 側に置く）

## 実装メモ

- `ignore` の判定は優先順位不要（FR-5.12）＝いずれかにマッチすれば除外。`Lookup` のパターンマッチ部分を切り出して共有すると良い。
- config の読み込みは `Requires` によるサブ Analyzer に切り出してもよい（DEC-11.2、同一 pass 内での重複読み込み回避）。ただし必須ではない。
- `pass.Module` は driver によっては nil（`go doc` で確認済み）。フォールバックの stderr メッセージ文言は英語・小文字始まり（DEC-11.9）。
- **書き込みは一切行わない**（FR-6.15 / DEC-1.4.1）。

## 終了条件

- [ ] `testdata/src/<pkg>` 直下に `importalias.json` を置いた analysistest で、config の alias 指定が多数決より優先されることを確認。
- [ ] `ignore` に該当する package が診断されないことを確認するテスト。
- [ ] `pass.Module == nil` 相当（config なし）でも従来どおり多数決のみで動作することを単体で確認（フォールバック）。
- [ ] `-strict` 有効時、複数 alias が存在する時点で tie 扱いになり診断が保留（自動確定しない）ことを確認。
- [ ] `go test ./...` 全 green。

## 依存

なし。02/03 と同じく analyzer.go に触れるため、それらと着手順を調整（config を渡す `decide.Decide` のシグネチャは 02 で変わる可能性があるので、02 の後だと差分が素直）。
