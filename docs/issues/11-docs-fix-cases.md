# 11: docs/fix-cases.md の新設と既存ケースの明文化

## 目的

`testdata/fix/<case>` と 1:1 で対応する auto-fix ユースケース仕様書 `docs/fix-cases.md` を新設する。既存の 6 ケースをまず明文化し、以降の fix issue（09/10）が「ケース追加 → 実装」ループで追記していく土台を確立する。

## 対象 DEC・FR

- DEC-7.2（fix テストは golden file 方式。各 `<case>` に対応するユースケース仕様を `docs/fix-cases.md` にケース名一致で記述し、「何のためのケースか」「どんな入力から何が期待されるか」を人間が読んで分かる状態にする）

## 変更ファイル

- `docs/fix-cases.md`（新規）

## 実装メモ

- 既存 `testdata/fix/` のケース一覧（ディレクトリ名＝ケース名で一致させる）:
  - `rename_to_majority` — 単一ファイル、多数決で定まった alias へ既存 alias を rename。
  - `multi_file_majority` — 複数ファイル。foo.go/bar.go が `f`（2 票）、boo.go が `oldf`（1 票）→ `f` に統合、boo.go だけ変更。
  - `collision_unaliased_to_alias` — 無 alias `fmt` を `f` に rename しようとするがローカル変数 `f` と衝突 → スキップ（無変更）。
  - `collision_alias_to_unaliased` — `f "fmt"` を無 alias 化しようとするがローカル変数 `fmt` と衝突 → スキップ（無変更）。
  - `no_collision_unrelated_scope` — 無関係な別スコープの同名変数は衝突ではない → fix 成功。
  - `unalias_no_collision` — `f "fmt"` を衝突なく無 alias 化 → fix 成功。
- 各ケースに「目的／入力の要点／期待される出力（変更 or 無変更）／根拠 FR・DEC」を記述する。
- 冒頭に「本書は testdata/fix と同期する。新しい fix ケースを足すときは、まずここにケースを書いてから testdata と実装を追加する」という運用宣言を書く（DEC-7.5 のループ）。

## 終了条件

- [ ] `docs/fix-cases.md` が存在し、既存 `testdata/fix/` の全ケースが漏れなく記載されている。
- [ ] 記載のケース名が `testdata/fix/` のディレクトリ名と完全一致している。
- [ ] 09/10 が追記していくための運用宣言が冒頭にある。

## 依存

なし（既存 testdata の明文化なので単独で着手可能。09/10 の前に closed にしておくのが望ましい）。
