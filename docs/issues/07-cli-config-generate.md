# 07: CLI の config 生成・更新（`-fix` なし既定動作）

## 目的

`-fix` を指定しない場合の CLI 既定動作である「スキャン結果を元に `importalias.json` を生成・更新する」を実装する。既存の `shape.Load`/`shape.Merge`/`shape.Save` を配線し、tie 発生箇所は候補配列として書き込む。

## 対象 DEC・FR

- FR-7.2〜7.4（決定ロジックで「正」を算出 → 副産物として設定ファイル生成・更新。tie は 2 候補以上を書き込む）
- FR-7.20（`-fix` なしの既定はスキャン＋設定生成のみ、ソース書き換えなし）
- DEC-11.8（スキャン範囲外の package キーは保持、範囲内は置換）
- DEC-11.13（範囲内 package は path 単位も含め丸ごと置換）
- FR-5.4 / DEC-2.1 / DEC-11.17（tie は `AliasValue.Tie` に 2 要素以上、昇順で表現）
- FR-7.19 / FR-5.8（`-config` フラグ、省略時 module root 直下 `importalias.json`）

## 変更ファイル

- `cmd/goimportalias/`（decide 結果から `shape.File`（fresh）を構築 → `shape.Load`（既存）→ `shape.Merge(existing, fresh)` → `shape.Save`。`-config` 対応。tie の Decision は `AliasValue.Tie` に、resolved は `AliasValue.Resolved` に落とす）

## 実装メモ

- fresh の `File.Packages[pkg][path]` に、各 Decision を `AliasValue` へ変換して詰める。`d.Tie` なら `Tie: d.TieCandidate`、そうでなければ `Resolved: d.WantAlias`。
- `shape.Merge` は DEC-11.8/11.13 の意味で既に実装済み。`ignore` は existing から保持される（Merge 実装済み挙動）。
- 出力フォーマットの安定性（インデント2スペース・末尾改行・キーのソート）は `shape.Save`/`MarshalJSON` が担保済み（DEC-2.2）。
- FR-6.11 の `AliasCollision` を設定ファイルにどう書くかは本 issue のスコープ外でよい（検出は 02、fix は 09）。まず FR-6.10 系の Decision を書ければ十分。

## 終了条件

- [ ] golden 方式のテスト: 「入力 testdata＋既存 importalias.json なし」→ 生成される importalias.json が期待どおり。
- [ ] 「既存 importalias.json あり」→ スキャン範囲外エントリが保持され（DEC-11.8）、範囲内 package は path 単位で丸ごと置換される（DEC-11.13）ことを確認。
- [ ] tie が 2 要素以上の配列として昇順で書き込まれることを確認。
- [ ] `-config` で出力先を明示指定でき、省略時は module root 直下 `importalias.json`。
- [ ] `-fix` なしではソースが書き換わらないことを確認（FR-7.20）。
- [ ] `go test ./...` 全 green。

## 依存

06（CLI スキャン・module root・go/packages）が closed であること。
