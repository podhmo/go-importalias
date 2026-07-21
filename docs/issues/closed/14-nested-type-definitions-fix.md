# 14: ネストした型定義内の scan / auto-fix 検証

## 目的

type alias、interface 型、struct 型のネストした型定義内に現れる qualified identifier でも、scan が import の利用箇所として検出し、既存の auto-fix が canonical alias へ安全に書き換えることを test / golden test で確認する。

## 対象 DEC・FR

- DEC-7.2（fix テストは golden file 方式）
- DEC-7.5（ケース追加 → 実装のループ）
- FR-6.10（同一 import path の alias 多数決と auto-fix）

## 変更ファイル

- `docs/fix-cases.md`
- `internal/fix/pipeline_test.go`
- `internal/scan/scan_test.go`
- `testdata/fix/nested_type_definitions/`

## 実装メモ

- 既存の `scan -> decide -> fix` pipeline test と同じ形で、複数ファイルの多数決により `fmt` を `f` へ rewrite する。
- 対象 fixture では、type alias の右辺にある interface/struct 型リテラルをネストし、その中の `fmt.Stringer`、`fmt.State` などの type-position selector が漏れなく `f.*` へ変わることを確認する。
- `internal/scan` 側では、同じ fixture の `UsePos` にネストした型定義内の qualified identifier がすべて入ることを確認する。

## 終了条件

- [ ] `testdata/fix/nested_type_definitions` の input/golden が `docs/fix-cases.md` に記載されている。
- [ ] scan が type alias、ネストした interface 型、ネストした struct 型内の qualified identifier を `UsePos` として検出する。
- [ ] type alias、ネストした interface 型、ネストした struct 型内の qualified identifier が golden output で canonical alias に書き換わる。
- [ ] `go test ./...` と `go vet ./...` が green。

## 依存

なし（既存 fix pipeline 上のカバレッジ追加として単独で着手可能）。
