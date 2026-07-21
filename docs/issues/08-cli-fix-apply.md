# 08: CLI `-fix`：auto-fix 適用とファイル書き戻し

## 目的

`-fix` フラグ指定時に、決定ロジックで一意に「正」が定まった不整合へ auto-fix を適用し、変更をファイルに書き戻す。既存の `internal/fix.ApplyToFile`（rename ケース＋衝突チェック）で足りる範囲を CLI から配線する。

## 対象 DEC・FR

- FR-7.5（一意に定まった自明な変換のみ auto-fix）
- FR-7.6（config に 2 候補並記＝tie 未解決の対象は auto-fix 対象外）
- FR-7.7（import 節＋同一ファイル内の qualified identifier を整合的に書き換え）
- FR-7.18（`-fix` フラグ）
- DEC-11.7（パッケージ毎逐次）

## 変更ファイル

- `cmd/goimportalias/`（`-fix` 指定時、パッケージ毎逐次に、各ファイルへ `fix.ApplyToFile(fset, file, info, decisions)` を呼び、`changed` なら書き戻す。修正内容を人間可読に出力。tie/衝突でスキップされた箇所も報告）

## 実装メモ

- `fix.ApplyToFile` は `(src []byte, changed bool, err error)` を返す。`changed` のファイルだけ `os.WriteFile` する。
- 型情報は 06 で用意した `go/packages` ロード結果（`types.Info`、`Scopes` 含む）をそのまま渡す。
- tie 未解決・衝突により自明変換でない箇所は `ApplyToFile` 側が既にスキップする（`changed=false`）。CLI はそれを「保留」として報告する。
- FR-6.11 / FR-6.16 の auto-fix はこの issue のスコープ外（09/10）。この issue は既存の rename ケース（FR-6.10・FR-7.21 の別名除去）を CLI から通せれば十分。
- 書き戻し後の gofmt 差分混入は許容（DEC-11.6）。

## 終了条件

- [ ] `-fix` 前後のソースを golden 比較する統合テスト（rename ケース＝既存 `internal/fix` の範囲）が green。
- [ ] tie/衝突ケースが書き換えられず、診断（保留）のみ残ることを確認。
- [ ] `-fix` で全不整合が解消したとき終了コード 0、残存があれば 1（DEC-4.4）。
- [ ] 複数ファイルにまたがるケース（負けた alias のファイルだけ書き換わる）が正しく動く。
- [ ] `go test ./...` 全 green。

## 依存

06（CLI スキャン）が closed であること。07（config 生成）とは独立だが、両方揃うと CLI として一通り完成する。
